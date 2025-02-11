#!/usr/bin/env python3

from opcua import ua, Server
from opcua.ua import uatypes

class Complex(uatypes.FrozenClass):
    ua_types = [
        ('i', 'Int64'),
        ('j', 'Int64'),
    ]

    def __init__(self, i=0, j=0):
        self.i = i
        self.j = j
        self._freeze = True

    def __str__(self):
        return f'Complex(i:{self.i}, j:{self.j})'

    __repr__ = __str__

def even(parent, variant):
    print("even method call with parameters: ", variant.Value)
    ret = (variant.Value % 2 == 0)
    print("even", type(ret))
    return [ua.Variant(ret, ua.VariantType.Boolean)]

def square(parent, variant):
    print("square method call with parameters: ", variant.Value)
    variant.Value *= variant.Value
    return [variant]

def sum_of_squares(parent, variant):
    v = variant.Value # type is extension object "Complex"
    print("sum_of_square method call with parameter: ", v)
    ret = v.i*v.i + v.j*v.j
    return [ua.Variant(ret, ua.VariantType.Int64)]

def issue_768(parent):
    print("no_args method call returns []ua.ExtensionObject")
    return [ua.Variant([Complex(1,2), Complex(3,4)])]

if __name__ == "__main__":
    server = Server()
    server.set_endpoint("opc.tcp://0.0.0.0:4840/")

    ns = server.register_namespace("http://gopcua.com/")
    uatypes.register_extension_object('Complex', ua.NodeId("ComplexType", ns), Complex)
    main = server.nodes.objects.add_object(ua.NodeId("main", ns), "main")
    fnEven = main.add_method(ua.NodeId("even", ns), "even", even, [ua.VariantType.Int64], [ua.VariantType.Boolean])
    fnSquare = main.add_method(ua.NodeId("square", ns), "square", square, [ua.VariantType.Int64], [ua.VariantType.Int64])
    fnSumOfSquare = main.add_method(ua.NodeId("sumOfSquare", ns), "sumOfSquare", sum_of_squares, [ua.VariantType.ExtensionObject], [ua.VariantType.Int64])
    fnNoArgs = main.add_method(ua.NodeId("issue768", ns), "issue768", issue_768, [])

    server.start()
