/*
This is a stub to brainstorm having multiple ways of distribution of images.
Currently having a registry, but future may hold a p2p/bittorrent style
distribution, or even a single remote URL of a tar of layers, or even a direct
communications to a peer docker engine.

While this could be initialized on daemon start, this may need to be available
on `docker pull` (i.e. daemon started with p2p, but there is a need to pull an
image in a registry).
*/
package distributiondriver

type Driver interface {
}
