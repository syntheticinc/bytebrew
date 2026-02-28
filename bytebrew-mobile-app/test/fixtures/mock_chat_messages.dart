import 'package:bytebrew_mobile/core/domain/ask_user.dart';
import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/plan.dart';
import 'package:bytebrew_mobile/core/domain/tool_call.dart';

/// Chat messages for session-2 (bytebrew-srv) -- rich conversation with
/// tool calls, reasoning, plan updates, and markdown.
final mockSession2Messages = [
  ChatMessage(
    id: 'msg-2-01',
    type: ChatMessageType.userMessage,
    content: 'Проанализируй auth модуль и предложи рефакторинг',
    timestamp: DateTime.now().subtract(const Duration(minutes: 10)),
  ),
  ChatMessage(
    id: 'msg-2-02',
    type: ChatMessageType.reasoning,
    content:
        'Пользователь хочет рефакторинг auth модуля. '
        'Нужно изучить текущую структуру, найти проблемы с SOLID...',
    timestamp: DateTime.now().subtract(const Duration(minutes: 9, seconds: 55)),
  ),
  ChatMessage(
    id: 'msg-2-03',
    type: ChatMessageType.agentMessage,
    content: 'Начинаю анализ auth модуля. Сначала изучу структуру проекта.',
    timestamp: DateTime.now().subtract(const Duration(minutes: 9, seconds: 50)),
  ),
  ChatMessage(
    id: 'msg-2-04',
    type: ChatMessageType.toolCall,
    content: '',
    timestamp: DateTime.now().subtract(const Duration(minutes: 9, seconds: 40)),
    toolCall: const ToolCallData(
      id: 'tc-2-01',
      toolName: 'search_code',
      arguments: {'query': 'auth', 'path': 'internal/'},
      status: ToolCallStatus.completed,
      result: 'Found 12 matches in 5 files',
      fullResult:
          'internal/auth/handler.go:12: func NewHandler\n'
          'internal/auth/handler.go:45: func (h *Handler) Login\n'
          'internal/auth/handler.go:89: func (h *Handler) Register\n'
          'internal/auth/service.go:15: type Service struct\n'
          'internal/auth/service.go:34: func (s *Service) Authenticate\n'
          'internal/auth/service.go:78: func (s *Service) HashPassword\n'
          'internal/auth/service.go:95: func (s *Service) VerifyPassword\n'
          'internal/auth/service.go:112: func (s *Service) GenerateToken\n'
          'internal/auth/service.go:145: func (s *Service) ValidateToken\n'
          'internal/auth/middleware.go:8: func AuthMiddleware\n'
          'internal/config/config.go:23: JWTSecret string\n'
          'internal/delivery/grpc/server.go:67: auth.NewHandler',
    ),
  ),
  ChatMessage(
    id: 'msg-2-05',
    type: ChatMessageType.toolCall,
    content: '',
    timestamp: DateTime.now().subtract(const Duration(minutes: 9, seconds: 20)),
    toolCall: const ToolCallData(
      id: 'tc-2-02',
      toolName: 'read_file',
      arguments: {'path': 'internal/auth/handler.go'},
      status: ToolCallStatus.completed,
      result: '245 lines',
      fullResult:
          'package auth\n\nimport (\n\t"context"\n\t"net/http"\n)\n\n'
          'type Handler struct {\n\tservice *Service\n}\n\n'
          'func NewHandler(service *Service) *Handler {\n'
          '\treturn &Handler{service: service}\n}\n\n'
          '// ... 245 lines of handler code',
    ),
  ),
  ChatMessage(
    id: 'msg-2-06',
    type: ChatMessageType.toolCall,
    content: '',
    timestamp: DateTime.now().subtract(const Duration(minutes: 9)),
    toolCall: const ToolCallData(
      id: 'tc-2-03',
      toolName: 'read_file',
      arguments: {'path': 'internal/auth/service.go'},
      status: ToolCallStatus.completed,
      result: '189 lines',
      fullResult:
          'package auth\n\nimport (\n\t"context"\n\t"crypto/bcrypt"\n'
          '\t"time"\n\n\t"github.com/golang-jwt/jwt/v5"\n)\n\n'
          'type Service struct {\n\tdb *gorm.DB\n\tjwtSecret string\n}\n\n'
          '// ... 189 lines of service code',
    ),
  ),
  ChatMessage(
    id: 'msg-2-07',
    type: ChatMessageType.agentMessage,
    content:
        '## Анализ Auth модуля\n\n'
        '### Текущая структура\n'
        '- `handler.go` (245 строк) — HTTP handlers\n'
        '- `service.go` (189 строк) — бизнес-логика\n\n'
        '### Проблемы\n'
        '1. **SRP нарушен** — Service содержит и валидацию, и JWT, '
        'и хеширование\n'
        '2. **Нет интерфейсов** — прямая зависимость от конкретных типов\n'
        '3. **handler.go > 200 строк** — триггер для проверки\n\n'
        '### Рекомендации\n\n'
        'Разделить на:\n\n'
        '```go\n'
        '// internal/usecase/auth/usecase.go\n'
        'type TokenGenerator interface {\n'
        '    Generate(ctx context.Context, userID string) (string, error)\n'
        '}\n'
        '\n'
        'type Usecase struct {\n'
        '    tokens TokenGenerator\n'
        '    hasher PasswordVerifier\n'
        '}\n'
        '```\n\n'
        '- `TokenService` — JWT операции\n'
        '- `PasswordHasher` — хеширование',
    timestamp: DateTime.now().subtract(const Duration(minutes: 8)),
  ),
  ChatMessage(
    id: 'msg-2-08',
    type: ChatMessageType.planUpdate,
    content: '',
    timestamp: DateTime.now().subtract(const Duration(minutes: 7, seconds: 30)),
    plan: PlanData(
      goal: 'Рефакторинг auth модуля',
      steps: [
        PlanStep(
          index: 0,
          description: 'Выделить TokenService интерфейс',
          status: PlanStepStatus.completed,
          completedAt: DateTime.now().subtract(const Duration(minutes: 6)),
        ),
        PlanStep(
          index: 1,
          description: 'Выделить PasswordHasher',
          status: PlanStepStatus.completed,
          completedAt: DateTime.now().subtract(const Duration(minutes: 4)),
        ),
        const PlanStep(
          index: 2,
          description: 'Создать AuthUsecase',
          status: PlanStepStatus.inProgress,
        ),
        const PlanStep(
          index: 3,
          description: 'Обновить handler с DI',
          status: PlanStepStatus.pending,
        ),
        const PlanStep(
          index: 4,
          description: 'Написать тесты',
          status: PlanStepStatus.pending,
        ),
      ],
    ),
  ),
  ChatMessage(
    id: 'msg-2-09',
    type: ChatMessageType.toolCall,
    content: '',
    timestamp: DateTime.now().subtract(const Duration(minutes: 3)),
    toolCall: const ToolCallData(
      id: 'tc-2-04',
      toolName: 'write_file',
      arguments: {'path': 'internal/usecase/auth/usecase.go'},
      status: ToolCallStatus.running,
    ),
  ),
  ChatMessage(
    id: 'msg-2-10',
    type: ChatMessageType.reasoning,
    content:
        'Создаю AuthUsecase с consumer-side интерфейсами. '
        'Нужно определить TokenGenerator и PasswordVerifier '
        'прямо в файле usecase...',
    timestamp: DateTime.now().subtract(const Duration(minutes: 2, seconds: 50)),
  ),
];

/// Chat messages for session-1 (api-gateway) -- short conversation with
/// a pending ask-user prompt.
final mockSession1Messages = [
  ChatMessage(
    id: 'msg-1-01',
    type: ChatMessageType.userMessage,
    content: 'Настрой аутентификацию для API gateway',
    timestamp: DateTime.now().subtract(const Duration(minutes: 5)),
  ),
  ChatMessage(
    id: 'msg-1-02',
    type: ChatMessageType.agentMessage,
    content:
        'Для API gateway нужно выбрать метод аутентификации. '
        'Есть два основных подхода:',
    timestamp: DateTime.now().subtract(const Duration(minutes: 4, seconds: 30)),
  ),
  ChatMessage(
    id: 'msg-1-03',
    type: ChatMessageType.askUser,
    content: '',
    timestamp: DateTime.now().subtract(const Duration(minutes: 4)),
    askUser: const AskUserData(
      id: 'ask-1-01',
      question: 'Какой метод аутентификации использовать?',
      options: ['JWT tokens', 'Session cookies'],
      status: AskUserStatus.pending,
    ),
  ),
];

/// All mock messages keyed by session ID.
final mockChatMessages = <String, List<ChatMessage>>{
  'session-1': mockSession1Messages,
  'session-2': mockSession2Messages,
};
