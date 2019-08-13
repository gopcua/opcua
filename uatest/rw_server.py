#!/usr/bin/env python3

import logging
from opcua import ua, Server

if __name__ == "__main__":
    logging.basicConfig(level=logging.WARN)

    server = Server()
    server.set_endpoint("opc.tcp://0.0.0.0:4840/gopcua/server/")
    server.set_server_name("OPC/UA server Read/Write tests")

    uri = "http://gopcua.com/"
    ns = server.register_namespace(uri)

    main = server.nodes.objects.add_object(ua.NodeId("main", ns), "main")
    roBool = main.add_variable(ua.NodeId("ro_bool", ns), "ro_bool", True, ua.VariantType.Boolean)
    rwBool = main.add_variable(ua.NodeId("rw_bool", ns), "rw_bool", True, ua.VariantType.Boolean)
    rwBool.set_writable()

    roInt32 = main.add_variable(ua.NodeId("ro_int32", ns), "ro_int32", 5, ua.VariantType.Int32)
    rwInt32 = main.add_variable(ua.NodeId("rw_int32", ns), "rw_int32", 5, ua.VariantType.Int32)
    rwInt32.set_writable()

    server.start()
