# dfinger

🎯 `dfinger` 是一款 Web 指纹识别与信息收集工具，支持多目标扫描、高并发、指纹匹配、favicon hash识别、标题提取等能力。

> 本项目仅供合法授权测试与学习交流使用，禁止用于非法用途。

##  功能特性

- ✅ 多协议自动识别（http/https）
- ✅ 内置精简优选端口（支持自定义）
- ✅ 网页标题提取 + 内容长度统计
- ✅ Favicon Hash 指纹识别
- ✅ Web 指纹识别（基于关键词匹配 + icon hash 等）
- ✅ 并发扫描 + 超时控制
- ✅ 支持结果保存
- ✅ 颜色输出，简洁红队风格日志格式

## 使用示例

```bash
# 基础用法
dfinger.exe -u http://example.com:8080/admin
dfinger.exe -u 111.2.3.4/24
dfinger.exe -u 111.2.3.4/24 -p 80,443,8080

# 多线程 + 超时时间控制
dfinger.exe -u http://example.com -t 200 -timeout 5

# 从文件中批量导入目标
dfinger.exe -f targets.txt

# 指定指纹文件（json类型）
dfinger.exe -f targets.txt -finger test.json
```

## 指纹编写

```json
{
  "cms": "指纹名称",
  "level": 1-5,（可信度等级  5最高）
  "logic": "and/or", （conditions的匹配逻辑  and 或者 or）
  "tags": ["标签1", "标签2"],  (简化的指纹名称，用于匹配POC)
  "conditions": [
    {
      "location": "body",(body/header/title/favicon/path五种匹配位置)
      "matcher": "match",（match/regex两种匹配方法）
      "keywords": ["关键词1", "关键词2"]（关键词采用的是and逻辑  必须全都配对才认为匹配成功）
    }，
   {
      "location": "header",
      "matcher": "regex",
      "keywords": ["关键词1", "关键词2"]
    },
   {
      "location": "favicon",(fofa的ico哈希）
      "matcher": "match",
      "keywords": ["-525659379"]
    }
  ]
}
```



