import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/app_colors.dart';
import '../application/auth_provider.dart';

/// Login / Register screen with email and password fields.
class LoginScreen extends ConsumerStatefulWidget {
  const LoginScreen({super.key});

  @override
  ConsumerState<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends ConsumerState<LoginScreen> {
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  bool _isRegisterMode = false;

  @override
  void dispose() {
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  void _submit() {
    final email = _emailController.text.trim();
    final password = _passwordController.text;

    if (email.isEmpty || password.isEmpty) return;

    final authNotifier = ref.read(authProvider.notifier);
    if (_isRegisterMode) {
      authNotifier.register(email, password);
    } else {
      authNotifier.login(email, password);
    }
  }

  @override
  Widget build(BuildContext context) {
    final authState = ref.watch(authProvider);
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;

    ref.listen<AuthState>(authProvider, (previous, next) {
      if (next.status == AuthStatus.authenticated) {
        context.go('/sessions');
        return;
      }

      if (next.status == AuthStatus.error && next.error != null) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(next.error!)));
      }
    });

    final isLoading = authState.status == AuthStatus.loading;

    return Scaffold(
      backgroundColor: isDark ? AppColors.dark : AppColors.light,
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.symmetric(horizontal: 32),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                _Wordmark(theme: theme, isDark: isDark),
                const SizedBox(height: 48),
                _EmailField(controller: _emailController, enabled: !isLoading),
                const SizedBox(height: 16),
                _PasswordField(
                  controller: _passwordController,
                  enabled: !isLoading,
                  onSubmitted: (_) => _submit(),
                ),
                const SizedBox(height: 8),
                if (authState.status == AuthStatus.error &&
                    authState.error != null)
                  _ErrorText(message: authState.error!),
                const SizedBox(height: 24),
                _SubmitButton(
                  isLoading: isLoading,
                  isRegisterMode: _isRegisterMode,
                  onPressed: _submit,
                ),
                const SizedBox(height: 16),
                _ModeToggle(
                  isRegisterMode: _isRegisterMode,
                  onToggle: () {
                    setState(() => _isRegisterMode = !_isRegisterMode);
                  },
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

/// Brand wordmark with subtitle.
class _Wordmark extends StatelessWidget {
  const _Wordmark({required this.theme, required this.isDark});

  final ThemeData theme;
  final bool isDark;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Text(
          'Byte Brew',
          style: theme.textTheme.headlineLarge?.copyWith(
            fontWeight: FontWeight.bold,
            color: isDark ? AppColors.light : AppColors.dark,
            letterSpacing: -0.5,
          ),
        ),
        const SizedBox(height: 8),
        Text(
          'Sign in to continue',
          style: theme.textTheme.bodyMedium?.copyWith(
            color: AppColors.shade3,
            letterSpacing: 0.5,
          ),
        ),
      ],
    );
  }
}

/// Email text field.
class _EmailField extends StatelessWidget {
  const _EmailField({required this.controller, required this.enabled});

  final TextEditingController controller;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    return TextField(
      key: const ValueKey('email_field'),
      controller: controller,
      enabled: enabled,
      keyboardType: TextInputType.emailAddress,
      autocorrect: false,
      textInputAction: TextInputAction.next,
      decoration: const InputDecoration(
        labelText: 'Email',
        prefixIcon: Icon(Icons.email_outlined),
      ),
    );
  }
}

/// Password text field with obscured input.
class _PasswordField extends StatelessWidget {
  const _PasswordField({
    required this.controller,
    required this.enabled,
    required this.onSubmitted,
  });

  final TextEditingController controller;
  final bool enabled;
  final ValueChanged<String> onSubmitted;

  @override
  Widget build(BuildContext context) {
    return TextField(
      key: const ValueKey('password_field'),
      controller: controller,
      enabled: enabled,
      obscureText: true,
      textInputAction: TextInputAction.done,
      decoration: const InputDecoration(
        labelText: 'Password',
        prefixIcon: Icon(Icons.lock_outlined),
      ),
      onSubmitted: onSubmitted,
    );
  }
}

/// Inline error message shown below the form fields.
class _ErrorText extends StatelessWidget {
  const _ErrorText({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(top: 8),
      child: Text(
        message,
        style: Theme.of(context).textTheme.bodySmall?.copyWith(
          color: Theme.of(context).colorScheme.error,
        ),
        textAlign: TextAlign.center,
      ),
    );
  }
}

/// Full-width submit button with loading state.
class _SubmitButton extends StatelessWidget {
  const _SubmitButton({
    required this.isLoading,
    required this.isRegisterMode,
    required this.onPressed,
  });

  final bool isLoading;
  final bool isRegisterMode;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: double.infinity,
      height: 48,
      child: FilledButton(
        onPressed: isLoading ? null : onPressed,
        child: isLoading
            ? const SizedBox(
                width: 24,
                height: 24,
                child: CircularProgressIndicator(
                  strokeWidth: 2,
                  color: AppColors.light,
                ),
              )
            : Text(isRegisterMode ? 'Create Account' : 'Sign In'),
      ),
    );
  }
}

/// Toggle between login and register modes.
class _ModeToggle extends StatelessWidget {
  const _ModeToggle({required this.isRegisterMode, required this.onToggle});

  final bool isRegisterMode;
  final VoidCallback onToggle;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return TextButton(
      onPressed: onToggle,
      child: Text(
        isRegisterMode
            ? 'Already have an account? Sign in'
            : "Don't have an account? Register",
        style: theme.textTheme.bodySmall?.copyWith(color: AppColors.accent),
      ),
    );
  }
}
