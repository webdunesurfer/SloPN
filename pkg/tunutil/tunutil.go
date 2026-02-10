package tunutil

type Config struct {
	Name            string
	Addr            string
	Peer            string
	Mask            string
	MTU             int
	SkipSubnetRoute bool
	NoRoute         bool // If true, do not touch the routing table at all
}
