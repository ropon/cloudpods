# Requirements Document

## Introduction

本功能旨在为 CloudPods 块存储系统新增 iSCSI 存储类型支持。iSCSI (Internet Small Computer Systems Interface) 是一种基于 IP 网络的存储协议，允许通过网络连接到远程存储设备。通过添加 iSCSI 存储类型，用户可以在创建块存储时选择 iSCSI 作为存储后端，从而支持更多样化的存储基础设施。

## Requirements

### Requirement 1

**User Story:** 作为系统管理员，我希望能够在创建块存储时选择 iSCSI 作为存储类型，以便可以使用基于 iSCSI 协议的存储设备。

#### Acceptance Criteria

1. WHEN 用户在"新建块存储"对话框中查看存储类型选项 THEN 系统 SHALL 在存储类型列表中显示 "iSCSI" 选项
2. WHEN 用户选择 iSCSI 存储类型 THEN 系统 SHALL 显示 iSCSI 相关的配置字段
3. WHEN 用户选择 iSCSI 存储类型 THEN 系统 SHALL 隐藏其他存储类型特有的配置字段

### Requirement 2

**User Story:** 作为系统管理员，我希望能够配置 iSCSI 连接参数，以便系统能够正确连接到 iSCSI 目标设备。

#### Acceptance Criteria

1. WHEN 用户选择 iSCSI 存储类型 THEN 系统 SHALL 显示 "iSCSI Target" 输入字段用于配置目标地址
2. WHEN 用户选择 iSCSI 存储类型 THEN 系统 SHALL 显示 "iSCSI IQN" 输入字段用于配置 iSCSI 限定名称
3. WHEN 用户选择 iSCSI 存储类型 THEN 系统 SHALL 显示 "Portal" 输入字段用于配置 iSCSI 门户地址
4. WHEN 用户选择 iSCSI 存储类型 THEN 系统 SHALL 显示可选的认证字段（用户名和密码）
5. WHEN 用户输入无效的 IQN 格式 THEN 系统 SHALL 显示格式错误提示

### Requirement 3

**User Story:** 作为系统管理员，我希望系统能够验证 iSCSI 连接配置的有效性，以便在创建存储前确保配置正确。

#### Acceptance Criteria

1. WHEN 用户填写完 iSCSI 配置信息并点击确定 THEN 系统 SHALL 验证 iSCSI Target 地址的可达性
2. WHEN iSCSI 连接验证失败 THEN 系统 SHALL 显示具体的错误信息
3. WHEN iSCSI 连接验证成功 THEN 系统 SHALL 允许创建存储
4. WHEN 用户提供了认证信息 THEN 系统 SHALL 验证认证凭据的有效性

### Requirement 4

**User Story:** 作为开发人员，我希望 iSCSI 存储类型能够与现有的存储管理系统集成，以便提供一致的存储操作体验。

#### Acceptance Criteria

1. WHEN 创建 iSCSI 类型的块存储 THEN 系统 SHALL 使用统一的存储管理接口
2. WHEN 对 iSCSI 存储执行挂载操作 THEN 系统 SHALL 正确处理 iSCSI 协议的连接和挂载
3. WHEN 对 iSCSI 存储执行卸载操作 THEN 系统 SHALL 正确断开 iSCSI 连接
4. WHEN 删除 iSCSI 类型的块存储 THEN 系统 SHALL 清理相关的 iSCSI 连接资源

### Requirement 5

**User Story:** 作为系统管理员，我希望能够监控和管理 iSCSI 存储的状态，以便及时发现和处理存储问题。

#### Acceptance Criteria

1. WHEN 查看块存储列表 THEN 系统 SHALL 正确显示 iSCSI 存储的状态信息
2. WHEN iSCSI 连接出现问题 THEN 系统 SHALL 在存储状态中反映连接异常
3. WHEN iSCSI 存储正常工作 THEN 系统 SHALL 显示"在线"或"可用"状态
4. WHEN 系统检测到 iSCSI 存储异常 THEN 系统 SHALL 记录相关日志信息