# Implementation Plan

- [x] 1. 添加 iSCSI 存储类型常量和 API 结构定义

  - 在 pkg/apis/compute/storage_const.go 中添加 STORAGE_ISCSI 常量
  - 在 pkg/apis/compute/storage.go 中扩展 StorageCreateInput 结构，添加 iSCSI 相关字段
  - 创建 iSCSI 存储配置结构定义
  - Requirements: 1.1, 2.1, 2.2, 2.3, 2.4

- [x] 2. 实现 iSCSI 存储驱动核心逻辑

  - 创建 pkg/compute/storagedrivers/iscsi.go 文件
  - 实现 SIscsiStorageDriver 结构体，继承 SBaseStorageDriver
  - 实现 GetStorageType() 方法返回 STORAGE_ISCSI
  - 在 init() 函数中注册 iSCSI 存储驱动
  - Requirements: 4.1, 4.2

- [x] 3. 实现 iSCSI 配置验证逻辑

  - 在 SIscsiStorageDriver 中实现 ValidateCreateData() 方法
  - 添加 IQN 格式验证函数，确保符合 RFC 3720 标准
  - 添加 Target 和 Portal 地址格式验证
  - 实现 iSCSI 连接可用性测试逻辑
  - 添加认证参数验证（当提供用户名密码时）
  - Requirements: 2.5, 3.1, 3.2, 3.4

- [x] 4. 实现存储创建后处理逻辑

  - 在 SIscsiStorageDriver 中实现 PostCreate() 方法
  - 创建存储缓存配置
  - 将 iSCSI 配置参数序列化并存储到 StorageConf 字段
  - 设置存储的默认参数和状态
  - Requirements: 4.1

- [x] 5. 扩展存储更新功能支持 iSCSI

  - 在 SIscsiStorageDriver 中实现 ValidateUpdateData() 方法
  - 支持更新 iSCSI 认证信息
  - 实现配置变更的验证逻辑
  - 添加配置更新后的连接测试
  - Requirements: 3.1, 3.2

- [x] 6. 实现主机端 iSCSI 存储管理器

  - 在 pkg/hostman/storageman 目录下创建 iSCSI 存储管理器
  - 实现 SIscsiStorageManager 结构体
  - 实现 MountStorage() 方法，使用 iscsiadm 工具连接 iSCSI target
  - 实现 UnmountStorage() 方法，断开 iSCSI 连接并清理资源
  - 添加 iSCSI 设备发现和路径管理逻辑
  - Requirements: 4.2, 4.3

- [ ] 7. 添加 iSCSI 存储状态监控

  - 实现 iSCSI 连接状态检查逻辑
  - 在存储状态更新中集成 iSCSI 连接状态
  - 添加连接异常时的状态标记逻辑
  - 实现定期连接状态检查机制
  - Requirements: 5.1, 5.2, 5.3

- [ ] 8. 实现错误处理和日志记录

  - 添加 iSCSI 特定的错误类型定义
  - 实现详细的错误信息返回逻辑
  - 添加 iSCSI 操作的日志记录
  - 实现连接失败时的重试机制
  - Requirements: 3.2, 5.4

- [ ] 9. 创建 iSCSI 存储驱动单元测试

  - 为 SIscsiStorageDriver 创建单元测试文件
  - 测试 IQN 格式验证功能
  - 测试 Target 地址验证功能
  - 测试配置参数验证逻辑
  - 测试存储创建和更新流程
  - Requirements: 2.5, 3.1, 3.4

- [ ] 10. 创建主机端 iSCSI 管理器测试

  - 为 iSCSI 存储管理器创建单元测试
  - 模拟 iscsiadm 命令执行和测试挂载逻辑
  - 测试设备发现和路径管理功能
  - 测试错误处理和恢复机制
  - Requirements: 4.2, 4.3

- [x] 11. 实现前端 UI 支持

  - 在存储创建表单中添加 iSCSI 存储类型选项
  - 添加 iSCSI Target 地址输入字段
  - 添加 iSCSI IQN 输入字段
  - 添加 iSCSI Portal 地址输入字段
  - 添加可选的认证信息输入字段（用户名/密码）
  - 添加 LUN ID 输入字段（默认值为 0）
  - Requirements: 1.1, 1.2, 1.3

- [x] 12. 添加前端表单验证

  - 实现前端 IQN 格式验证
  - 实现 IP 地址和端口格式验证
  - 添加必填字段验证
  - 实现实时配置验证反馈
  - Requirements: 2.5, 1.3

- [ ] 13. 集成测试和端到端验证
  - 创建完整的 iSCSI 存储创建流程测试
  - 测试存储挂载到主机的完整流程
  - 验证存储状态监控功能
  - 测试错误场景的处理
  - 验证前端和后端的集成
  - Requirements: 1.1, 2.1, 3.1, 4.1, 5.1
