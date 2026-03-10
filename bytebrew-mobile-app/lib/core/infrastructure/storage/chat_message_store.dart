import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:sqflite/sqflite.dart';

import 'package:bytebrew_mobile/core/domain/ask_user.dart';
import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/plan.dart';
import 'package:bytebrew_mobile/core/domain/tool_call.dart';

/// Persistent SQLite storage for chat messages.
///
/// Provides CRUD operations for [ChatMessage] entities, serializing complex
/// fields (toolCall, askUser, plan) as JSON strings. Uses a singleton database
/// instance initialized lazily on first access.
class ChatMessageStore {
  ChatMessageStore({String? dbPath}) : _dbPath = dbPath;

  final String? _dbPath;
  Database? _db;

  static const _tableName = 'chat_messages';
  static const _dbFileName = 'bytebrew_chat.db';

  /// Returns the singleton database, creating it on first call.
  Future<Database> _getDb() async {
    if (_db != null) return _db!;

    final path = _dbPath ?? '${await getDatabasesPath()}/$_dbFileName';
    _db = await openDatabase(
      path,
      version: 1,
      onCreate: (db, version) async {
        await db.execute('''
          CREATE TABLE IF NOT EXISTS $_tableName (
            id TEXT PRIMARY KEY,
            session_id TEXT NOT NULL,
            type TEXT NOT NULL,
            content TEXT NOT NULL,
            timestamp INTEGER NOT NULL,
            agent_id TEXT,
            tool_call TEXT,
            ask_user TEXT,
            plan TEXT
          )
        ''');
        await db.execute('''
          CREATE INDEX IF NOT EXISTS idx_session_timestamp
          ON $_tableName (session_id, timestamp)
        ''');
      },
    );
    return _db!;
  }

  /// Retrieves messages for a session, ordered by timestamp ascending.
  Future<List<ChatMessage>> getMessages(String sessionId, {int? limit}) async {
    final db = await _getDb();
    final rows = await db.query(
      _tableName,
      where: 'session_id = ?',
      whereArgs: [sessionId],
      orderBy: 'timestamp ASC',
      limit: limit,
    );
    return rows.map(_rowToMessage).toList();
  }

  /// Inserts or replaces a single message.
  Future<void> upsertMessage(String sessionId, ChatMessage message) async {
    final db = await _getDb();
    await db.insert(
      _tableName,
      _messageToRow(sessionId, message),
      conflictAlgorithm: ConflictAlgorithm.replace,
    );
  }

  /// Inserts or replaces multiple messages in a single transaction.
  Future<void> upsertMessages(
    String sessionId,
    List<ChatMessage> messages,
  ) async {
    if (messages.isEmpty) return;

    final db = await _getDb();
    final batch = db.batch();
    for (final message in messages) {
      batch.insert(
        _tableName,
        _messageToRow(sessionId, message),
        conflictAlgorithm: ConflictAlgorithm.replace,
      );
    }
    await batch.commit(noResult: true);
  }

  /// Deletes all messages for a session.
  Future<void> deleteSession(String sessionId) async {
    final db = await _getDb();
    await db.delete(
      _tableName,
      where: 'session_id = ?',
      whereArgs: [sessionId],
    );
  }

  /// Closes the database. Call when the app is shutting down.
  Future<void> close() async {
    await _db?.close();
    _db = null;
  }

  // ---------------------------------------------------------------------------
  // Serialization
  // ---------------------------------------------------------------------------

  Map<String, Object?> _messageToRow(String sessionId, ChatMessage message) {
    return {
      'id': message.id,
      'session_id': sessionId,
      'type': message.type.name,
      'content': message.content,
      'timestamp': message.timestamp.millisecondsSinceEpoch,
      'agent_id': message.agentId,
      'tool_call': message.toolCall != null
          ? jsonEncode(_toolCallToJson(message.toolCall!))
          : null,
      'ask_user': message.askUser != null
          ? jsonEncode(_askUserToJson(message.askUser!))
          : null,
      'plan': message.plan != null
          ? jsonEncode(_planToJson(message.plan!))
          : null,
    };
  }

  ChatMessage _rowToMessage(Map<String, Object?> row) {
    final typeStr = row['type'] as String;
    final type = ChatMessageType.values.firstWhere(
      (t) => t.name == typeStr,
      orElse: () => ChatMessageType.systemMessage,
    );

    return ChatMessage(
      id: row['id'] as String,
      type: type,
      content: row['content'] as String,
      timestamp: DateTime.fromMillisecondsSinceEpoch(row['timestamp'] as int),
      agentId: row['agent_id'] as String?,
      toolCall: _toolCallFromJson(row['tool_call'] as String?),
      askUser: _askUserFromJson(row['ask_user'] as String?),
      plan: _planFromJson(row['plan'] as String?),
    );
  }

  // -- ToolCallData --

  Map<String, Object?> _toolCallToJson(ToolCallData data) {
    return {
      'id': data.id,
      'toolName': data.toolName,
      'arguments': data.arguments,
      'status': data.status.name,
      'result': data.result,
      'fullResult': data.fullResult,
      'error': data.error,
    };
  }

  ToolCallData? _toolCallFromJson(String? json) {
    if (json == null) return null;
    try {
      final map = jsonDecode(json) as Map<String, dynamic>;
      return ToolCallData(
        id: map['id'] as String,
        toolName: map['toolName'] as String,
        arguments: (map['arguments'] as Map<String, dynamic>).map(
          (k, v) => MapEntry(k, v as String),
        ),
        status: ToolCallStatus.values.firstWhere(
          (s) => s.name == map['status'],
          orElse: () => ToolCallStatus.running,
        ),
        result: map['result'] as String?,
        fullResult: map['fullResult'] as String?,
        error: map['error'] as String?,
      );
    } catch (e) {
      debugPrint('[ChatMessageStore] Failed to parse toolCall: $e');
      return null;
    }
  }

  // -- AskUserData --

  Map<String, Object?> _askUserToJson(AskUserData data) {
    return {
      'id': data.id,
      'question': data.question,
      'options': data.options,
      'status': data.status.name,
      'answer': data.answer,
    };
  }

  AskUserData? _askUserFromJson(String? json) {
    if (json == null) return null;
    try {
      final map = jsonDecode(json) as Map<String, dynamic>;
      return AskUserData(
        id: map['id'] as String,
        question: map['question'] as String,
        options: (map['options'] as List<dynamic>).cast<String>(),
        status: AskUserStatus.values.firstWhere(
          (s) => s.name == map['status'],
          orElse: () => AskUserStatus.pending,
        ),
        answer: map['answer'] as String?,
      );
    } catch (e) {
      debugPrint('[ChatMessageStore] Failed to parse askUser: $e');
      return null;
    }
  }

  // -- PlanData --

  Map<String, Object?> _planToJson(PlanData data) {
    return {
      'goal': data.goal,
      'steps': [
        for (final step in data.steps)
          {
            'index': step.index,
            'description': step.description,
            'status': step.status.name,
            'completedAt': step.completedAt?.millisecondsSinceEpoch,
          },
      ],
    };
  }

  PlanData? _planFromJson(String? json) {
    if (json == null) return null;
    try {
      final map = jsonDecode(json) as Map<String, dynamic>;
      final stepsList = (map['steps'] as List<dynamic>)
          .cast<Map<String, dynamic>>();
      return PlanData(
        goal: map['goal'] as String,
        steps: [
          for (final s in stepsList)
            PlanStep(
              index: s['index'] as int,
              description: s['description'] as String,
              status: PlanStepStatus.values.firstWhere(
                (st) => st.name == s['status'],
                orElse: () => PlanStepStatus.pending,
              ),
              completedAt: s['completedAt'] != null
                  ? DateTime.fromMillisecondsSinceEpoch(s['completedAt'] as int)
                  : null,
            ),
        ],
      );
    } catch (e) {
      debugPrint('[ChatMessageStore] Failed to parse plan: $e');
      return null;
    }
  }
}
