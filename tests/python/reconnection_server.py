#!/usr/bin/env python3

from opcua import ua, Server
from opcua.common.callback import CallbackType
from opcua.ua.ua_binary import struct_to_binary, header_to_binary

server = None

# HACK: opcua does not support sending error message
def send_error_message(status_code, reason="test bench simulation"):
  # take the first transport they all have the same processor
  transport = server.iserver.asyncio_transports[0]
  processor = transport._protocol.processor

  response = ua.ErrorMessage()
  response.Error = status_code
  response.Reason = reason

  processor.socket.write(error_message_to_binary(response))

# HACK: opcua does not support sending error message
def error_message_to_binary(message):
  header = ua.Header(ua.MessageType.Error, ua.ChunkType.Single)
  body = struct_to_binary(message)
  header.body_size = len(body)
  return header_to_binary(header) + body

# On connection_lost the server remove the securechannel and the session
def simulate_connection_failure(parent):

  def close_connection():
    for transport in server.iserver.asyncio_transports:
      transport.close()

  server.iserver.loop.call_soon(close_connection)
  return []

def simulate_securechannel_failure(parent):
  server.iserver.loop.call_soon(lambda : send_error_message(ua.StatusCode(ua.StatusCodes.BadSecureChannelIdInvalid)))
  return []

def simulate_session_failure(parent):
  server.iserver.loop.call_soon(lambda : send_error_message(ua.StatusCode(ua.StatusCodes.BadSessionIdInvalid)))
  return []

def simulate_subscription_failure(parent):
  server.iserver.loop.call_soon(lambda : send_error_message(ua.StatusCode(ua.StatusCodes.BadSubscriptionIdInvalid)))
  return []

if __name__ == "__main__":
  server = Server()
  server.set_endpoint("opc.tcp://0.0.0.0:4840/")
  ns = server.register_namespace("http://gopcua.com/")

  simulations = server.nodes.objects.add_object(ua.NodeId("simulations", ns), "simulations")
  fnEven = simulations.add_method(ua.NodeId("simulate_connection_failure", ns), "simulate_connection_failure", simulate_connection_failure, [], [])
  fnEven = simulations.add_method(ua.NodeId("simulate_securechannel_failure", ns), "simulate_securechannel_failure", simulate_securechannel_failure, [], [])
  fnEven = simulations.add_method(ua.NodeId("simulate_session_failure", ns), "simulate_session_failure", simulate_session_failure, [], [])
  fnEven = simulations.add_method(ua.NodeId("simulate_subscription_failure", ns), "simulate_subscription_failure", simulate_subscription_failure, [], [])

  server.start()
