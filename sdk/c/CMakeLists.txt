cmake_minimum_required(VERSION 3.14)
project(eipc_c C)

set(CMAKE_C_STANDARD 11)
set(CMAKE_C_STANDARD_REQUIRED ON)

# EIPC C SDK static library
add_library(eipc_c STATIC
    src/eipc_client.c
    src/eipc_server.c
    src/eipc_transport.c
    src/eipc_frame.c
    src/eipc_hmac.c
    src/eipc_sha256.c
    src/eipc_json.c
)

target_include_directories(eipc_c PUBLIC
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
    $<INSTALL_INTERFACE:include>
)

target_include_directories(eipc_c PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/src
)

if(WIN32)
    target_link_libraries(eipc_c PUBLIC ws2_32)
endif()

if(MSVC)
    target_compile_definitions(eipc_c PRIVATE _CRT_SECURE_NO_WARNINGS)
endif()

# ── eipc_easy convenience layer ──
add_library(eipc_easy STATIC src/eipc_easy.c)
target_link_libraries(eipc_easy PUBLIC eipc_c)

if(MSVC)
    target_compile_definitions(eipc_easy PRIVATE _CRT_SECURE_NO_WARNINGS)
endif()

# ── ALIAS targets (same names as exported namespace) ──
add_library(eipc::eipc ALIAS eipc_c)
add_library(eipc::easy ALIAS eipc_easy)

# ── Install & Package Config ──
if(NOT EIPC_SKIP_INSTALL)
    include(GNUInstallDirs)
    include(CMakePackageConfigHelpers)

    install(TARGETS eipc_c eipc_easy
        EXPORT eipcTargets
        ARCHIVE  DESTINATION ${CMAKE_INSTALL_LIBDIR}
        LIBRARY  DESTINATION ${CMAKE_INSTALL_LIBDIR}
        RUNTIME  DESTINATION ${CMAKE_INSTALL_BINDIR}
    )

    install(DIRECTORY include/
        DESTINATION ${CMAKE_INSTALL_INCLUDEDIR}
    )

    install(EXPORT eipcTargets
        FILE        eipcTargets.cmake
        NAMESPACE   eipc::
        DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/eipc
    )

    configure_package_config_file(
        ${CMAKE_CURRENT_SOURCE_DIR}/cmake/eipcConfig.cmake.in
        ${CMAKE_CURRENT_BINARY_DIR}/eipcConfig.cmake
        INSTALL_DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/eipc
    )

    write_basic_package_version_file(
        ${CMAKE_CURRENT_BINARY_DIR}/eipcConfigVersion.cmake
        VERSION 1.0.0
        COMPATIBILITY SameMajorVersion
    )

    install(FILES
        ${CMAKE_CURRENT_BINARY_DIR}/eipcConfig.cmake
        ${CMAKE_CURRENT_BINARY_DIR}/eipcConfigVersion.cmake
        DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/eipc
    )
endif()

# Tests
if(BUILD_TESTING OR EIPC_BUILD_TESTS)
    enable_testing()
    add_subdirectory(tests)
endif()
