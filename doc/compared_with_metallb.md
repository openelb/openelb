# Compared with MetalLB

## Pros
- Support most BGP features. They are implemented by gobgp
- Configured through CRD. Bgp global  and Bgp neighbor could add/update/delete dynamically.
- Passive mode is provided. Porter Manager could connect router voluntarily.
- Use DNAT to handle non-bgp port, could coexist with calico etc.
                                        
## Cons
 - Support Linux only

## Similarity
- Need more test