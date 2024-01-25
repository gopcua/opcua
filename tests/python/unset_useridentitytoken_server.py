from opcua import Server, ua

if __name__ == "__main__":
  # Create a server instance
  server = Server()

  # Set the endpoint
  server.set_endpoint("opc.tcp://localhost:4840")
  ns = server.register_namespace("http://gopcua.com/")
  simulations = server.nodes.objects.add_object(ua.NodeId("simulations", ns), "simulations")

  # Create a UserTokenPolicy
  policy = ua.UserTokenPolicy()
  policy.TokenType = ua.UserTokenType.UserName
  policy.PolicyId = "username"

  # Add the policy to the server
  server.set_security_policy([ua.SecurityPolicyType.NoSecurity])
  
  # Start the server
  server.start()
