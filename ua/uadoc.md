# ActivateSessionRequest

Specification: Part 4, 5.6.3.2

ActivateSessionRequest is used by the Client to specify the identity of the user
associated with the Session. This Service request shall be issued by the Client
before it issues any Service request other than CloseSession after CreateSession.
Failure to do so shall cause the Server to close the Session.

Whenever the Client calls this Service the Client shall prove that it is the same application that
called the CreateSession Service. The Client does this by creating a signature with the private key
associated with the clientCertificate specified in the CreateSession request. This signature is
created by appending the last serverNonce provided by the Server to the serverCertificate and
calculating the signature of the resulting sequence of bytes.


# ActivateSessionResponse

Specification: Part 4, 5.6.3.2

ActivateSessionResponse is used by the Server to answer to the ActivateSessionRequest.
Once used, a serverNonce cannot be used again. For that reason, the Server returns a new
serverNonce each time the ActivateSession Service is called.

When the ActivateSession Service is called for the first time then the Server shall reject the
request if the SecureChannel is not same as the one associated with the CreateSession request.
Subsequent calls to ActivateSession may be associated with different SecureChannels. If this is
the case then the Server shall verify that the Certificate the Client used to create the new
SecureChannel is the same as the Certificate used to create the original SecureChannel. In
addition, the Server shall verify that the Client supplied a UserIdentityToken that is identical to the
token currently associated with the Session. Once the Server accepts the new SecureChannel it
shall reject requests sent via the old SecureChannel.

# ApplicationType

Specification: Part 4, 7.1

# ApplicationDescription

Specification: Part 4, 7.1
