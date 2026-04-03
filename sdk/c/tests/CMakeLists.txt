cmake_minimum_required(VERSION 3.10)

# EIPC C SDK Tests
add_executable(test_hmac test_hmac.c)
target_link_libraries(test_hmac eipc_c)
add_test(NAME test_hmac COMMAND test_hmac)

add_executable(test_frame test_frame.c)
target_link_libraries(test_frame eipc_c)
add_test(NAME test_frame COMMAND test_frame)

add_executable(test_transport test_transport.c)
target_link_libraries(test_transport eipc_c)
add_test(NAME test_transport COMMAND test_transport)

add_executable(test_chat_json test_chat_json.c)
target_link_libraries(test_chat_json eipc_c)
add_test(NAME test_chat_json COMMAND test_chat_json)

add_executable(test_eipc_easy test_eipc_easy.c)
target_link_libraries(test_eipc_easy eipc_easy)
add_test(NAME test_eipc_easy COMMAND test_eipc_easy)
