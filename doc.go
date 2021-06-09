/*
Package cocoa provides interfaces to interact with groups of containers (called
pods) backed by container orchestration services. Containers are not managed
individually - they're managed as logical groupings of containers.

The ECSPodCreator interface provides an abstraction to create pods backed by ECS
without needing to make direct calls to the API.

The ECSPod is a self-contained unit that allows users to manage their pod
without having to make direct calls to the API. It is backed by an ECSClient.

The ECSClient interface provides a convenience wrapper around the ECS API. If
the ECSPodCreator and ECSPod does not fulfill your needs, you can make API calls
directly to ECS instead.
*/
package cocoa
