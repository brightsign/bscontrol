# Cloud API SDK for Go

## Project Overview

This is a Go SDK that provides idiomatic, type-safe access to a cloud API service. The SDK abstracts the complexity of HTTP requests, authentication, retries, and error handling, allowing developers to interact with the cloud service using clean Go interfaces.

### Purpose

- Provide a simple, intuitive Go API for cloud service operations
- Handle authentication, rate limiting, and retries automatically
- Offer strong typing and compile-time safety
- Support both synchronous and streaming operations

# Instructions

* be brief - don't sound like an AI - be more human and summarize
* explain things like a Senior Software Engineer communicating with peers
* always provide block diagrams and sequence diagrams and use mermaid diagrams whenever possible
* always have a .gitignore file, and always have *~ and ./external in it
* ./external is for information that you don't need to keep in context unless I ask
* if you are unsure about anything, stop and ask for more clarity
* you are likely working inside a container with no access to files parallel or above your working directory
 - therefore don't try to find things outside the folder you are in and subdirectories
*

## Go Related Instructions

* this is a project written in golang
* this is a go module, always structure things expecting it to be "required" in other projects
* create a folder layout according to go best practices
* append lines to the .gitignore file according to go best practices, keeping other lines already present
* prefer to use tagged go modules via a "require" statement in preference to using "replace"
 - this applies unless otherwise instructred


## Testing Related Instructions

* we will always design and implement tests for all code we write
* tests should NEVER be skipped, especially if they are not passing
* if you ever think a test should be skipped, stop and ask for more clarity

## Project Related Instructions
