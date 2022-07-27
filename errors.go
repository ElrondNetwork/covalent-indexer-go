package covalent

import "errors"

// ErrNilPubKeyConverter signals that a pub key converter input parameter is nil
var ErrNilPubKeyConverter = errors.New("received nil input value: pub key converter")

// ErrBlockBodyAssertion signals that an error occurred when trying to assert BodyHandler interface of type block body
var ErrBlockBodyAssertion = errors.New("error asserting BodyHandler interface of type block body")

// ErrNilHasher signals that a nil hasher has been provided
var ErrNilHasher = errors.New("received nil input value: hasher")

// ErrNilMarshaller signals that a nil marshaller has been provided
var ErrNilMarshaller = errors.New("received nil input value: marshaller")

// ErrNilMiniBlockHandler signals that a nil mini block handler has been provided
var ErrNilMiniBlockHandler = errors.New("received nil input value: mini block handler")

// ErrNilShardCoordinator signals that a shard coordinator input parameter is nil
var ErrNilShardCoordinator = errors.New("received nil input value: shard coordinator")

// ErrNilDataHandler signals that a nil data handler handler has been provided
var ErrNilDataHandler = errors.New("received nil input value: data handler")

// ErrNilHTTPServer signals that a nil http server has been provided
var ErrNilHTTPServer = errors.New("received nil input value: http server")

// ErrNilAlteredAccounts signals that a nil altered accounts object has been provided
var ErrNilAlteredAccounts = errors.New("nil altered accounts")

// ErrAccountNotFound signals that a given account hasn't been found
var ErrAccountNotFound = errors.New("account not found")

// ErrCannotCreateBigIntFromString signals that a big int cannot be created from a string
var ErrCannotCreateBigIntFromString = errors.New("cannot create big int from string")
