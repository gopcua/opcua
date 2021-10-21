#!/usr/bin/env python3

from opcua import ua, Server

if __name__ == "__main__":
    server = Server()
    server.set_endpoint("opc.tcp://0.0.0.0:4840/")

    ns = server.register_namespace("http://gopcua.com/")
    main = server.nodes.objects.add_object(ua.NodeId("main", ns), "main")
    roBool = main.add_variable(ua.NodeId("ro_bool", ns), "ro_bool", True, ua.VariantType.Boolean)
    rwBool = main.add_variable(ua.NodeId("rw_bool", ns), "rw_bool", True, ua.VariantType.Boolean)
    rwBool.set_writable()

    roInt32 = main.add_variable(ua.NodeId("ro_int32", ns), "ro_int32", 5, ua.VariantType.Int32)
    rwInt32 = main.add_variable(ua.NodeId("rw_int32", ns), "rw_int32", 5, ua.VariantType.Int32)
    rwInt32.set_writable()

    roArrayInt32 = main.add_variable(ua.NodeId("array_int32", ns), "array_int32", [1,2,3], ua.VariantType.Int32)
    ro2DArrayInt32 = main.add_variable(ua.NodeId("2d_array_int32", ns), "2d_array_int32", [[1],[2],[3]], ua.VariantType.Int32)

    server.start()
