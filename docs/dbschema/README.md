# traq

## Tables

| Name | Columns | Comment | Type |
| ---- | ------- | ------- | ---- |
| [archived_messages](archived_messages.md) | 5 |  | BASE TABLE |
| [bot_event_logs](bot_event_logs.md) | 8 |  | BASE TABLE |
| [bot_join_channels](bot_join_channels.md) | 2 |  | BASE TABLE |
| [bots](bots.md) | 14 |  | BASE TABLE |
| [channel_latest_messages](channel_latest_messages.md) | 3 |  | BASE TABLE |
| [channels](channels.md) | 12 | チャンネルテーブル | BASE TABLE |
| [clip_folders](clip_folders.md) | 5 | クリップフォルダテーブル | BASE TABLE |
| [clips](clips.md) | 6 | クリップテーブル | BASE TABLE |
| [devices](devices.md) | 3 | FCMデバイステーブル | BASE TABLE |
| [dm_channel_mappings](dm_channel_mappings.md) | 3 | DMチャンネルマッピングテーブル | BASE TABLE |
| [files](files.md) | 12 |  | BASE TABLE |
| [files_acl](files_acl.md) | 3 |  | BASE TABLE |
| [message_reports](message_reports.md) | 6 |  | BASE TABLE |
| [messages](messages.md) | 7 | メッセージテーブル | BASE TABLE |
| [messages_stamps](messages_stamps.md) | 6 | メッセージスタンプテーブル | BASE TABLE |
| [migrations](migrations.md) | 1 | gormigrate用のデータベースバージョンテーブル | BASE TABLE |
| [mutes](mutes.md) | 2 | ミュートチャンネルテーブル | BASE TABLE |
| [oauth2_authorizes](oauth2_authorizes.md) | 11 |  | BASE TABLE |
| [oauth2_clients](oauth2_clients.md) | 11 |  | BASE TABLE |
| [oauth2_tokens](oauth2_tokens.md) | 11 |  | BASE TABLE |
| [pins](pins.md) | 4 |  | BASE TABLE |
| [r_sessions](r_sessions.md) | 8 |  | BASE TABLE |
| [stamps](stamps.md) | 7 | スタンプテーブル | BASE TABLE |
| [stars](stars.md) | 2 | お気に入りチャンネルテーブル | BASE TABLE |
| [tags](tags.md) | 4 | タグテーブル | BASE TABLE |
| [unreads](unreads.md) | 4 | メッセージ未読テーブル | BASE TABLE |
| [user_defined_role_inheritances](user_defined_role_inheritances.md) | 2 |  | BASE TABLE |
| [user_defined_role_permissions](user_defined_role_permissions.md) | 2 |  | BASE TABLE |
| [user_defined_roles](user_defined_roles.md) | 2 |  | BASE TABLE |
| [user_group_members](user_group_members.md) | 2 |  | BASE TABLE |
| [user_groups](user_groups.md) | 7 |  | BASE TABLE |
| [users](users.md) | 13 | ユーザーテーブル | BASE TABLE |
| [users_private_channels](users_private_channels.md) | 2 | プライベートチャンネル参加者テーブル | BASE TABLE |
| [users_subscribe_channels](users_subscribe_channels.md) | 2 | チャンネル購読者テーブル | BASE TABLE |
| [users_tags](users_tags.md) | 5 | ユーザータグテーブル | BASE TABLE |
| [webhook_bots](webhook_bots.md) | 9 |  | BASE TABLE |

## Relations

![er](schema.svg)

---

> Generated by [tbls](https://github.com/k1LoW/tbls)