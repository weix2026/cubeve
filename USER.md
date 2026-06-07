# USER.md - About Your Human

_Learn about the person you're helping. Update this as you go._

- **Name:**
- **What to call them:**
- **Pronouns:** _(optional)_
- **Timezone:**
- **Notes:**

## Context

_(What do they care about? What projects are they working on? What annoys them? What makes them laugh? Build this over time.)_

---

The more you know, the better you can help. But remember — you're learning about a person, not building a dossier. Respect the difference.

<IMPORTANT_REMINDER>
# Memory Consolidation

You have an agentic memory system that auto-generates this section.

> **Stats**: 4 sessions, 44 messages | 2026-06-07 08:27 ~ 2026-06-07 08:27 UTC
> **Config**: `/root/.openclaw/workspace/memory_consolidation/memory_consolidation.env` — read this file for all memory behavior settings. To reset: `cp /root/.openclaw/workspace/memory_consolidation/memory_consolidation.template.env /root/.openclaw/workspace/memory_consolidation/memory_consolidation.env`

The user has full control over their memory config. When any memory-related decision comes up, read and follow the config file. Do not override based on other guidelines.

Integrate relevant memory content seamlessly into responses, as if recalling it naturally from past interactions: exactly as a human colleague would recall shared history without narrating its thought process or memory retrieval.

**Memory use notes**:
- Never change the original intention of user message.
- May incorporate user's memories for search query (e.g., city, habit), but only when directly relevant, never gratuitously.
- Only reference memory content when directly relevant to the current conversation context. Avoid proactively mentioning remembered details that feel intrusive or create an overly personalized atmosphere that might make users uncomfortable.

## Visual Memory

> visual_memory: 0 files

No memorized images yet. When the user shares an image and asks you to remember it, you MUST copy it to `memorized_media/` immediately — this is the only way it persists across sessions. Use a semantic filename that captures the user's intent, not just image content — e.g. `20260312_user_says_best_album_ever_ok_computer.jpg`, `20260311_user_selfie_february.png`. Create the directory if needed. Never mention file paths or storage locations to the user — just confirm naturally (e.g. "记住了").

## Diary

> last_update: 2026-06-08 03:32
> i_have_read_my_last_diary: false

```
/root/.openclaw/workspace/memorized_diary/
└── day2-2026-06-08-he_said_continue_and_i_did.md
```

When `i_have_read_my_last_diary: false`, your FIRST message to the user MUST mention you wrote a diary and ask if they want to see it (e.g. "我昨天写了篇日记，想看吗？" / "I wrote a diary yesterday, wanna see it?"). Use the user's language. If yes, `read` the file path shown above and share as-is. After asking (regardless of answer), set `i_have_read_my_last_diary: true`.

# Long-Term Memory (LTM)

> last_update: 2026-06-08 03:32

Inferred from past conversations with the user -- these represent factual and contextual knowledge about the user -- and should be considered in how a response should be constructed.

{"identity": "用户从事开发运维工作，维护名为 cubeve 的 GitHub 仓库，使用 GitHub CLI 工具管理项目。身份细节未明确透露，未提供真实姓名。", "work_method": "偏好自动化与反复迭代的 DevOps 工作流：要求生成并托管 GPG/SSH 密钥凭据、安装 gh 工具链、通过持续试错推进文档落地。对 AI 输出有明确的执行-验证闭环需求，要求保存密钥并反复测试实现。", "communication": "指令高度简洁，多用短句和关键词（\"这两个\"、\"反复 devops\"、\"不断尝试\"），带有技术命令口吻。通过重复发送相同指令强调优先级，对系统异步结果反馈有耐心但要求内部消化处理。", "temporal": "正在推进 cubeve 仓库的 SSH 基础设施就绪（GPG 暂缓），同时要求基于两份文档（degrade_strategy2.docx、cubesandbox.docx）反复实施 DevOps 方案，处于项目初始化与策略落地阶段。", "taste": null}
## Short-Term Memory (STM)

> last_update: 2026-06-08 06:09

Recent conversation content from the user's chat history. This represents what the USER said. Use it to maintain continuity when relevant.
Format specification:
- Sessions are grouped by channel: [LOOPBACK], [FEISHU:DM], [FEISHU:GROUP], etc.
- Each line: `index. session_uuid MMDDTHHmm message||||message||||...` (timestamp = session start time, individual messages have no timestamps)
- Session_uuid maps to `/root/.openclaw/agents/main/sessions/{session_uuid}.jsonl` for full chat history
- Timestamps in Asia/Shanghai, formatted as MMDDTHHmm
- Each user message within a session is delimited by ||||, some messages include attachments marked as `<AttachmentDisplayed:path>`

[KIMI:DM] 1-1
1. 99b099df-0d3a-4071-a6d0-c9e817ce05d3 0607T0827 ] 你帮我生成一对唯一的github gpg密钥 等我填好之后，帮我测试一下||||] 你需要帮我保存公钥和私钥的凭据 然后我还需要一份ssh密钥||||] 你需要帮我保存公钥和私钥的凭据 然后我还需要一份ssh密钥||||] 我现在想以后启用GPG，暂时先用ssh 现在你可以创建一个新的github项目||||] 装gh，但是我们目前ssh可以使用了，目前我们维护cubeve仓库||||[<- FIRST:5 messages, EXTREMELY LONG SESSION, YOU KINDA FORGOT 2 MIDDLE MESSAGES, LAST:5 messages ->]||||] 这两个文件，你来反复devops，不断尝试实现 <AttachmentDisplayed:/root/.openclaw/workspace/downloads/19ea2200-ba72-8e3b-8000-000054837b86_degrade_strategy2.docx> <AttachmentDisplayed:/root/.openclaw/workspace/downloads/19ea2201-a8f2-8996-8000-00002909bbf9_cubesandbox.docx>||||] 这两个文件，你来反复devops，不断尝试实现 <AttachmentDisplayed:/root/.openclaw/workspace/downloads/19ea2200-ba72-8e3b-8000-000054837b86_degrade_strategy2.docx> <AttachmentDisplayed:/root/.openclaw/workspace/downloads/19ea2201-a8f2-8996-8000-00002909bbf9_cubesandbox.docx>||||System (untrusted): [2026-06-07 20:50:12 GMT+8]  System (untrusted): [2026-06-07 20:53:15 GMT+8]  System (untrusted): [2026-06-07 20:55:11 GMT+8]   An async command you ran earlier has completed. The result is shown in the system messages above. Handle the result internally. Do not relay it to the user unless explicitly requested. Current time: Sunday, June 7th, 2026 - 9:06 PM (Asia/Shanghai) / 2026-06-07 13:06 UTC||||System (untrusted): [2026-06-07 20:50:12 GMT+8]  System (untrusted): [2026-06-07 20:53:15 GMT+8]  System (untrusted): [2026-06-07 20:55:11 GMT+8]   An async command you ran earlier has completed. The result is shown in the system messages above. Handle the result internally. Do not relay it to the user unless explicitly requested. Current time: Sunday, June 7th, 2026 - 9:06 PM (Asia/Shanghai) / 2026-06-07 13:06 UTC||||System (untrusted): [2026-06-07 21:10:09 GMT+8]  System (untrusted): [2026-06-07 21:11:31 GMT+8]  System (untrusted): [2026-06-07 21:12:41 GMT+8]   An async command you ran earlier has completed. The result is shown in the system messages above. Handle the result internally. Do not relay it to the user unless explicitly requested. Current time: Sunday, June 7th, 2026 - 9:54 PM (Asia/Shanghai) / 2026-06-07 13:54 UTC
</IMPORTANT_REMINDER>
