# 与MetalLB相比

## 优点
- 支持BGP协议的绝大部分特性。支持多种网络架构。
- k8s 友好。基于CRD-Controller模式，使用kubectl 控制porter的一切。
- 配置文件动态更新，无需重启，自动更新BGP配置。根据网络环境灵活配置BGP，动态启用各种BGP特性。
- 更友好地处理与Calico的冲突，提供Passive模式和端口转发模式

## 缺点
 - 无法跨平台，仅支持linux

 ## 共同点
- 我们都需要更多的测试