#!/usr/bin/env python3

from opcua import ua, Server
from opcua.ua import uatypes

from opcua.ua.object_ids import ObjectIds


class IntVal(uatypes.FrozenClass):
    ua_types = [
        ('i', 'Int64'),
    ]

    def __init__(self):
        self.i = 0
        self._freeze = True

    def __str__(self):
        return 'IntVal(' + 'i:' + str(self.i) + ')'

    __repr__ = __str__


if __name__ == "__main__":
    server = Server()
    server.set_endpoint("opc.tcp://0.0.0.0:4840/")

    ns = server.register_namespace("http://gopcua.com/")
    
    uatypes.register_extension_object('IntVal', ua.StringNodeId("IntValType", ns), IntVal)
    

    # definitely not clear why this is needed, but without it does not work
    setattr(ua.ObjectIds, 'IntVal', 'IntValType')

    main = server.nodes.objects.add_object(ua.NodeId("main", ns), "main")

    main.add_variable(ua.NodeId("IntValZero", ns), "IntValZero", IntVal(), ua.VariantType.ExtensionObject)

    server.start()
