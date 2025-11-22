package main

//import cmpb "lab_3/proto/common/cmpb"

func mergeVectorClock(dst, src map[string]int32) {
	if dst == nil {
		dst = make(map[string]int32)
	}
	for node, val := range src {
		if cur, ok := dst[node]; !ok || val > cur {
			dst[node] = val
		}
	}
}

func copyVectorClock(vc map[string]int32) map[string]int32 {
	out := make(map[string]int32, len(vc))
	for k, v := range vc {
		out[k] = v
	}
	return out
}

func isBehind(local, client map[string]int32) bool {
	for k, cv := range client {
		lv := local[k]
		if lv < cv {
			return true
		}
	}
	return false
}
