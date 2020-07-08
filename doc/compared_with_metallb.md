# Compared with MetalLB

## Pros
- Support most BGP features and multiple network architectures.
- A Kubernetes-friendly tool based on CRD-Controller that can be controlled entirely by kubectl.
- The configuration file can be updated dynamically without any restart. BGP configurations are automatically updated based on the network environment. Various BGP features can be dynamically adopted.
- Provide Passive mode and support DNAT.
- Conflicts with Calico can be handled in a more friendly way.
                                        
## Cons
 - Support Linux only.

## Similarity
- More tests are needed for both tools.