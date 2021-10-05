#!/usr/bin/env python3

from opcua import ua, Server
from opcua.ua import uatypes

from opcua.ua.object_ids import ObjectIds


class Complex(uatypes.FrozenClass):
    ua_types = [
        ('i', 'Int64'),
        ('j', 'Int64'),
    ]

    def __init__(self):
        self.i = 0
        self.j = 0
        self._freeze = True

    def __str__(self):
        return 'Complex(' + 'i:' + str(self.i) + ', ' + \
               'j:' + str(self.j) + ')'

    __repr__ = __str__


if __name__ == "__main__":
    server = Server()
    server.set_endpoint("opc.tcp://0.0.0.0:4840/")

    ns = server.register_namespace("http://gopcua.com/")

    complexNode = ua.StringNodeId("ComplexType", ns)
    uatypes.register_extension_object('Complex', complexNode, Complex)

    # definitely not clear why this is needed, but without it does not work
    setattr(ua.ObjectIds, 'Complex', 'ComplexType')

    main = server.nodes.objects.add_object(ua.NodeId("main", ns), "main")

    complexZero = Complex()
    complexZero = main.add_variable(
        ua.NodeId("ComplexZero", ns), "ComplexZero", complexZero, ua.VariantType.ExtensionObject)

    server.start()
