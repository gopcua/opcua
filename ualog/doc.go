/*
Package ualog works as a facade towards the underlying logging framework.

It provides the opcua library with convenient structured logging, without exposing
the actual underlying logging framework to the library code.

# Basics

The ualog logger lives within the [context.Context] that is passed between functions
and is initialized by calling the ualog.New constructor that allows the instance to
be configured using the option pattern. If a logger isn't explicitly created, logging
will be performed using the current system default logger.
*/
package ualog
