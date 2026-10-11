// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

// Zero-copy tensor descriptors (Track 1 — tightly-coupled AI).
//
// A TensorDescriptor is a small, copyable handle that names a tensor's
// storage inside a transport/shm shared-memory segment. The descriptor
// itself travels over the ordinary eIPC control path (a Frame, or a
// core.Message Payload); the tensor payload never does — the consumer
// maps the named segment and reads [Offset, Offset+Size) directly.
//
// The data-type enumeration mirrors eAI's eai_dtype_t numeric values
// (eAI accel/include/eai/accel.h): identical values mean descriptors can
// be built from, and consumed by, eAI tensors without translation. Note
// that EAI_DTYPE_Q4_1 (6) and EAI_DTYPE_INT32 (8) exist in eAI but are
// not transported here yet; they are rejected by Validate and Decode.
package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// TensorDtype is the element data type of a tensor.
// Numeric values mirror eAI's eai_dtype_t.
type TensorDtype uint8

const (
	TensorF32  TensorDtype = 0 // EAI_DTYPE_F32
	TensorF16  TensorDtype = 1 // EAI_DTYPE_F16
	TensorBF16 TensorDtype = 2 // EAI_DTYPE_BF16
	TensorINT8 TensorDtype = 3 // EAI_DTYPE_INT8
	TensorINT4 TensorDtype = 4 // EAI_DTYPE_INT4
	TensorQ4_0 TensorDtype = 5 // EAI_DTYPE_Q4_0
	TensorQ8_0 TensorDtype = 7 // EAI_DTYPE_Q8_0 (eAI's Q4_1 is 6)
)

// MaxTensorDims mirrors EAI_TENSOR_MAX_DIMS.
const MaxTensorDims = 8

// TensorDescriptorSizeLimits bounds variable-length fields so a malicious
// peer cannot force an unbounded allocation on decode.
const (
	MaxTensorShmNameLen = 256
	MaxTensorCaps       = 32
	MaxTensorCapLen     = 128
	MaxTensorDigestLen  = 128
)

// dtypeBytes returns the element-level byte size used for stride arithmetic,
// mirroring eAI's eai_dtype_size (quantized dtypes count 1 byte per element
// at the stride level; packing is the producer/consumer's concern).
func (t TensorDtype) dtypeBytes() int64 {
	switch t {
	case TensorF32:
		return 4
	case TensorF16, TensorBF16:
		return 2
	case TensorINT8, TensorINT4, TensorQ4_0, TensorQ8_0:
		return 1
	}
	return 0
}

// TensorDescriptor names one tensor's storage in a shared-memory segment.
//
// Wire format (all multi-byte integers big-endian, Frame codec style):
//
//	[dtype:1][ndim:1][shape: ndim*8 int64][strides: ndim*8 int64]
//	[shm_name_len:2][shm_name bytes][offset:8][size:8]
//	[ncap:2]{[cap_len:2][cap bytes]}[digest_len:2][digest bytes]
type TensorDescriptor struct {
	Dtype        TensorDtype `json:"dtype"`
	Shape        []int64     `json:"shape"`
	Strides      []int64     `json:"strides"`
	ShmName      string      `json:"shm_name"`               // transport/shm segment name — the zero-copy handle
	Offset       uint64      `json:"offset"`                 // byte offset into the segment
	Size         uint64      `json:"size"`                   // byte span claimed in the segment
	Capabilities []string    `json:"capabilities,omitempty"` // #162-envelope capability claims, e.g. "inference.local"
	ModelDigest  string      `json:"model_digest,omitempty"` // sha256 of the signed model producing/consuming
}

var (
	ErrTensorDtype    = errors.New("eipc: unknown tensor dtype")
	ErrTensorDims     = errors.New("eipc: invalid tensor dimensions")
	ErrTensorStrides  = errors.New("eipc: strides inconsistent with shape/size")
	ErrTensorShmName  = errors.New("eipc: shm segment name missing for nonzero tensor size")
	ErrTensorDigest   = errors.New("eipc: model digest is not a valid sha256")
	ErrTensorCaps     = errors.New("eipc: too many or oversized capability claims")
	ErrTensorTooLarge = errors.New("eipc: tensor descriptor exceeds decode bounds")
)

// Validate checks the descriptor for internal consistency without touching
// shared memory: strides/size consistency, no zero dims, a shm name whenever
// a byte span is claimed, and a well-formed model digest.
//
// Size semantics: the claimed span [Offset, Offset+Size) must be at least
// as large as the tensor footprint implied by Shape and Strides, i.e.
// (1 + Σ (shape[i]-1)*strides[i]) * dtypeBytes. Padding beyond the
// footprint is allowed; a descriptor that claims less than its footprint
// would let a reader walk off the end of its mapping, so it is rejected.
func (d *TensorDescriptor) Validate() error {
	if d.Dtype.dtypeBytes() == 0 {
		return fmt.Errorf("%w: %d", ErrTensorDtype, uint8(d.Dtype))
	}
	ndim := len(d.Shape)
	if ndim == 0 || ndim > MaxTensorDims {
		return fmt.Errorf("%w: ndim=%d", ErrTensorDims, ndim)
	}
	for i, dim := range d.Shape {
		if dim <= 0 {
			return fmt.Errorf("%w: shape[%d]=%d (zero or negative dim)", ErrTensorDims, i, dim)
		}
	}
	if len(d.Strides) != ndim {
		return fmt.Errorf("%w: %d strides for %d dims", ErrTensorStrides, len(d.Strides), ndim)
	}
	elem := d.Dtype.dtypeBytes()
	var extent int64 = 1 // highest element index touched, 0-based
	for i := range d.Shape {
		s := d.Strides[i]
		if s < 0 {
			return fmt.Errorf("%w: strides[%d]=%d (negative stride)", ErrTensorStrides, i, s)
		}
		// Guard the multiply against int64 overflow.
		if d.Shape[i]-1 > 0 && s > (1<<62)/(d.Shape[i]-1) {
			return fmt.Errorf("%w: strides[%d] overflows extent", ErrTensorStrides, i)
		}
		extent += (d.Shape[i] - 1) * s
		if extent > (1<<62)/elem {
			return fmt.Errorf("%w: footprint overflows int64", ErrTensorStrides)
		}
	}
	footprint := uint64(extent) * uint64(elem)
	if d.Size < footprint {
		return fmt.Errorf("%w: size %d < footprint %d", ErrTensorStrides, d.Size, footprint)
	}
	if d.Size > 0 && d.ShmName == "" {
		return ErrTensorShmName
	}
	if len(d.ShmName) > MaxTensorShmNameLen {
		return fmt.Errorf("%w: %d bytes", ErrTensorShmName, len(d.ShmName))
	}
	if len(d.Capabilities) > MaxTensorCaps {
		return fmt.Errorf("%w: %d claims", ErrTensorCaps, len(d.Capabilities))
	}
	for i, c := range d.Capabilities {
		if c == "" || len(c) > MaxTensorCapLen {
			return fmt.Errorf("%w: capability[%d]", ErrTensorCaps, i)
		}
	}
	if d.ModelDigest != "" && !isSHA256Hex(d.ModelDigest) && !isSHA256Prefixed(d.ModelDigest) {
		return ErrTensorDigest
	}
	return nil
}

func isSHA256Hex(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func isSHA256Prefixed(s string) bool {
	const prefix = "sha256:"
	return len(s) == len(prefix)+64 && s[:len(prefix)] == prefix && isSHA256Hex(s[len(prefix):])
}

// Encode writes the descriptor to w in the wire format documented above.
func (d *TensorDescriptor) Encode(w io.Writer) error {
	ndim := len(d.Shape)
	if ndim == 0 || ndim > MaxTensorDims {
		return fmt.Errorf("%w: ndim=%d", ErrTensorDims, ndim)
	}
	if len(d.Strides) != ndim {
		return fmt.Errorf("%w: %d strides for %d dims", ErrTensorStrides, len(d.Strides), ndim)
	}

	buf := make([]byte, 0, 64+ndim*16+len(d.ShmName)+len(d.ModelDigest))
	buf = append(buf, byte(d.Dtype), byte(ndim))
	var tmp [8]byte
	for _, v := range d.Shape {
		binary.BigEndian.PutUint64(tmp[:], uint64(v))
		buf = append(buf, tmp[:]...)
	}
	for _, v := range d.Strides {
		binary.BigEndian.PutUint64(tmp[:], uint64(v))
		buf = append(buf, tmp[:]...)
	}
	buf = appendStringField(buf, d.ShmName)
	binary.BigEndian.PutUint64(tmp[:], d.Offset)
	buf = append(buf, tmp[:]...)
	binary.BigEndian.PutUint64(tmp[:], d.Size)
	buf = append(buf, tmp[:]...)
	if len(d.Capabilities) > MaxTensorCaps {
		return fmt.Errorf("%w: %d claims", ErrTensorCaps, len(d.Capabilities))
	}
	var ncap [2]byte
	binary.BigEndian.PutUint16(ncap[:], uint16(len(d.Capabilities)))
	buf = append(buf, ncap[:]...)
	for _, c := range d.Capabilities {
		buf = appendStringField(buf, c)
	}
	buf = appendStringField(buf, d.ModelDigest)

	_, err := w.Write(buf)
	return err
}

func appendStringField(buf []byte, s string) []byte {
	var ln [2]byte
	binary.BigEndian.PutUint16(ln[:], uint16(len(s)))
	buf = append(buf, ln[:]...)
	return append(buf, s...)
}

// DecodeTensor reads a descriptor from r. It enforces the decode bounds in
// TensorDescriptorSizeLimits so a truncated or hostile stream fails fast
// instead of allocating. Semantic checks (strides, dims, shm name) belong
// to Validate, which the caller should run before mapping anything.
func DecodeTensor(r io.Reader) (*TensorDescriptor, error) {
	head := make([]byte, 2)
	if _, err := io.ReadFull(r, head); err != nil {
		return nil, fmt.Errorf("read dtype/ndim: %w", err)
	}
	d := &TensorDescriptor{Dtype: TensorDtype(head[0])}
	ndim := int(head[1])
	if ndim == 0 || ndim > MaxTensorDims {
		return nil, fmt.Errorf("%w: ndim=%d", ErrTensorTooLarge, ndim)
	}

	readInt64s := func(n int) ([]int64, error) {
		raw := make([]byte, n*8)
		if _, err := io.ReadFull(r, raw); err != nil {
			return nil, err
		}
		out := make([]int64, n)
		for i := range out {
			out[i] = int64(binary.BigEndian.Uint64(raw[i*8 : (i+1)*8]))
		}
		return out, nil
	}

	var err error
	if d.Shape, err = readInt64s(ndim); err != nil {
		return nil, fmt.Errorf("read shape: %w", err)
	}
	if d.Strides, err = readInt64s(ndim); err != nil {
		return nil, fmt.Errorf("read strides: %w", err)
	}
	if d.ShmName, err = readStringField(r, MaxTensorShmNameLen, "shm_name"); err != nil {
		return nil, err
	}

	var offSize [16]byte
	if _, err := io.ReadFull(r, offSize[:]); err != nil {
		return nil, fmt.Errorf("read offset/size: %w", err)
	}
	d.Offset = binary.BigEndian.Uint64(offSize[0:8])
	d.Size = binary.BigEndian.Uint64(offSize[8:16])

	var ncapBuf [2]byte
	if _, err := io.ReadFull(r, ncapBuf[:]); err != nil {
		return nil, fmt.Errorf("read capability count: %w", err)
	}
	ncap := int(binary.BigEndian.Uint16(ncapBuf[:]))
	if ncap > MaxTensorCaps {
		return nil, fmt.Errorf("%w: %d capabilities", ErrTensorTooLarge, ncap)
	}
	if ncap > 0 {
		d.Capabilities = make([]string, 0, ncap)
		for i := 0; i < ncap; i++ {
			c, err := readStringField(r, MaxTensorCapLen, "capability")
			if err != nil {
				return nil, err
			}
			d.Capabilities = append(d.Capabilities, c)
		}
	}
	if d.ModelDigest, err = readStringField(r, MaxTensorDigestLen, "model_digest"); err != nil {
		return nil, err
	}
	return d, nil
}

func readStringField(r io.Reader, max int, what string) (string, error) {
	var ln [2]byte
	if _, err := io.ReadFull(r, ln[:]); err != nil {
		return "", fmt.Errorf("read %s length: %w", what, err)
	}
	n := int(binary.BigEndian.Uint16(ln[:]))
	if n > max {
		return "", fmt.Errorf("%w: %s length %d", ErrTensorTooLarge, what, n)
	}
	if n == 0 {
		return "", nil
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", fmt.Errorf("read %s: %w", what, err)
	}
	return string(buf), nil
}
