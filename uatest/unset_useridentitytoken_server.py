from opcua import Server, ua
from opcua.server.user_manager import UserManager

users = {'user': 'pass'}

def user_manager(isession, username, password):
  isession.user = UserManager.User
  return username in users and password == users[username]

if __name__ == "__main__":
  
  # Create a server instance
  server = Server()

  # Set the endpoint
  server.set_endpoint("opc.tcp://localhost:4840")
  ns = server.register_namespace("http://gopcua.com/")
  simulations = server.nodes.objects.add_object(ua.NodeId("simulations", ns), "simulations")

  # Add the policy to the server
  server.set_security_policy([ua.SecurityPolicyType.NoSecurity])
  server.set_security_IDs(["Username"])
  server.user_manager.user_manager = user_manager
  # Start the server
  server.start()
