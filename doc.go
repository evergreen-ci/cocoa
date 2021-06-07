/*
Package cocoa provides interfaces to interact with groups of containers (called
pods) backed by container orchestration services. Containers are not managed
individually - they're managed as logical groupings of containers.

The ECSPodManager interface provides an abstraction to interact with pods backed
by ECS without needing to make direct calls to the API to perform
frequently-used operations.

The ECSClient interface provides a convenience wrapper around the ECS API. If
the ECSPodManager does not fulfill your needs, you can make API calls directly
to ECS instead.

*/
package cocoa
