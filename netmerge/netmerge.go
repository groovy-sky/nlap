package netmerge

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"strings"
)

type IPv4Vector struct {
	FirstIP uint32
	LastIP  uint32
	CIDR    net.IPNet
}

// Parses input string and stores first IP, last IP and CIDR to IPv4Vector type
func cidrToVector(cidr string) (vector IPv4Vector, err error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return vector, err
	}

	// Convert IP address to uint32
	ipUint := binary.BigEndian.Uint32(ip.To4())

	// Calculate the last IP address in the CIDR block
	mask := binary.BigEndian.Uint32(ipNet.Mask)
	lastIP := (ipUint & mask) | (mask ^ 0xffffffff)

	vector = IPv4Vector{ipUint, lastIP, *ipNet}
	return vector, err
}

// Searches for closest IP ranges
func closestVectors(in *[]IPv4Vector) (closest1, closest2 IPv4Vector, err error) {

	var firstIndex, lastIndex int
	if len(*in) < 2 {
		return IPv4Vector{}, IPv4Vector{}, fmt.Errorf("[ERR]: Vectors number should be >2")
	}

	closestDist := math.Inf(1)

	for i := 0; i < len(*in)-1; i++ {
		for j := i + 1; j < len(*in); j++ {
			dist := float64(distance((*in)[i], (*in)[j]))
			if dist < closestDist {
				closest1 = (*in)[i]
				closest2 = (*in)[j]
				closestDist = dist
				if i > j {
					firstIndex = j
					lastIndex = i
				} else {
					firstIndex = i
					lastIndex = j
				}

			}
		}
	}

	*in = append((*in)[:lastIndex], (*in)[lastIndex+1:]...)
	*in = append((*in)[:firstIndex], (*in)[firstIndex+1:]...)

	return closest1, closest2, nil
}

// Calculate distance between two Vectors
func distance(v1, v2 IPv4Vector) uint32 {
	var minIP, maxIP uint32
	if v1.FirstIP > v2.FirstIP {
		minIP = v1.FirstIP - v2.FirstIP
	} else {
		minIP = v2.FirstIP - v1.FirstIP
	}

	if v1.LastIP > v2.LastIP {
		maxIP = v1.LastIP - v2.LastIP
	} else {
		maxIP = v2.LastIP - v1.LastIP
	}
	return uint32(float64(int32(minIP))) +
		uint32(float64(int32(maxIP)))
}

// Searches for a smallest a largest IPs in uint32 and tries to calulate mask for them
func mergeIPNets(v1, v2 *IPv4Vector) (out IPv4Vector, err error) {
	var minIP, maxIP uint32
	if v1.FirstIP > v2.FirstIP {
		minIP = v2.FirstIP
	} else {
		minIP = v1.FirstIP
	}

	if v1.LastIP > v2.LastIP {
		maxIP = v1.LastIP
	} else {
		maxIP = v2.LastIP
	}

	newMask := 32 - countDifferentBits(minIP, maxIP)
	newIP := binaryToIP(minIP).To4()
	newCIDR := fmt.Sprintf("%s/%d", newIP, newMask)

	out, err = cidrToVector(newCIDR)

	return out, err
}

// Search for a first different bit, starting from higher bit
func countDifferentBits(num1, num2 uint32) int {
	// Convert the numbers to binary strings
	bin1 := fmt.Sprintf("%032b", num1)
	bin2 := fmt.Sprintf("%032b", num2)

	// Compare the binary strings bit by bit
	for i := 0; i < 32; i++ {
		if bin1[i] != bin2[i] {
			// Count the number of bits left till the end of the input
			return 32 - i
		}
	}

	// If all bits match, return 0
	return 0
}

// Converts uint32 to net.IP format string
func binaryToIP(ip uint32) net.IP {
	return net.IPv4(byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
}

// Converts uint32 to IPv4 format string
func uint32ToIP(ip uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d", byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
}

// Merges input CIDRs to specified maxIpNum value
func MergeCIDRs(input []string, maxIpNum uint8) (out []string, err error) {
	var vectors []IPv4Vector
	for _, i := range input {
		if !strings.Contains(i, "/") {
			out = append(out, i)
		} else {
			v, err := cidrToVector(i)
			if err != nil {
				return []string{}, err
			} else {
				vectors = append(vectors, v)
			}
		}
	}

	var newRange IPv4Vector
	v1, v2, err := closestVectors(&vectors)
	if err != nil {
		return []string{}, err
	}
	newRange, err = mergeIPNets(&v1, &v2)
	if err != nil {
		return []string{}, err
	}
	vectors = append(vectors, newRange)

	for _, v := range vectors {
		ip := uint32ToIP(v.FirstIP)
		mask, _ := v.CIDR.Mask.Size()
		out = append(out, fmt.Sprintf("%s/%d", ip, mask))
	}
	return out, err
}
