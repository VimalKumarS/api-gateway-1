# api-gateway
API gateway that can be used as middleware to hide microservices behind, modify requests-responses or investigate interactions between client and server App. Written in Golang

## Troubleshooting
- returned redirects from server can be handled incorrectly - responses will be followed if they return redirect without a way to stop this behavior. Golang has trouble for now with handling this kind of requests - no flags available
