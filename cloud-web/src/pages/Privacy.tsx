export function PrivacyPage() {
  return (
    <div className="max-w-3xl mx-auto px-4 py-10">
      <h1 className="text-2xl font-bold text-brand-light">Privacy Policy</h1>
      <p className="mt-2 text-sm text-brand-shade2">Last modified: March 2026</p>

      <div className="mt-8 space-y-8 text-sm text-brand-shade2 leading-relaxed">
        <div>
          <h2 className="text-lg font-semibold text-brand-light">1. Introduction</h2>
          <p className="mt-2">
            ByteBrew (&quot;we&quot;, &quot;us&quot;, &quot;our&quot;) operates bytebrew.ai. This
            policy explains how we collect, use, and protect your personal information.
          </p>
        </div>

        <div>
          <h2 className="text-lg font-semibold text-brand-light">2. Information We Collect</h2>
          <ul className="mt-2 list-disc list-inside space-y-1 text-brand-shade3">
            <li>Account data: email address, hashed password</li>
            <li>Payment data: processed by Stripe (we don&apos;t store card numbers)</li>
            <li>Usage metadata: login timestamps, feature usage counts</li>
            <li>Technical data: IP address, browser type (for security)</li>
          </ul>
        </div>

        <div>
          <h2 className="text-lg font-semibold text-brand-light">3. How We Use Your Data</h2>
          <ul className="mt-2 list-disc list-inside space-y-1 text-brand-shade3">
            <li>Provide and maintain our services</li>
            <li>Process payments and manage subscriptions</li>
            <li>Send transactional emails (password reset, billing)</li>
            <li>Improve our products and services</li>
            <li>Prevent fraud and abuse</li>
          </ul>
        </div>

        <div>
          <h2 className="text-lg font-semibold text-brand-light">4. Data Sharing</h2>
          <p className="mt-2">We share data only with:</p>
          <ul className="mt-2 list-disc list-inside space-y-1 text-brand-shade3">
            <li>Stripe (payment processing)</li>
            <li>Resend (transactional email delivery)</li>
          </ul>
          <p className="mt-2 font-medium text-brand-light">
            We never sell your data to third parties.
          </p>
        </div>

        <div>
          <h2 className="text-lg font-semibold text-brand-light">5. Data Retention</h2>
          <ul className="mt-2 list-disc list-inside space-y-1 text-brand-shade3">
            <li>Active accounts: data stored while account exists</li>
            <li>Deleted accounts: personal data purged within 30 days</li>
            <li>Billing records: retained as required by law</li>
            <li>Anonymized usage data may be retained indefinitely</li>
          </ul>
        </div>

        <div>
          <h2 className="text-lg font-semibold text-brand-light">6. Your Rights</h2>
          <p className="mt-2">You have the right to:</p>
          <ul className="mt-2 list-disc list-inside space-y-1 text-brand-shade3">
            <li>Access your personal data</li>
            <li>Request deletion of your account and data</li>
            <li>Export your data</li>
            <li>Opt out of non-essential communications</li>
          </ul>
          <p className="mt-2">
            Contact us at{' '}
            <a
              href="mailto:privacy@bytebrew.ai"
              className="text-brand-accent hover:text-brand-accent-hover"
            >
              privacy@bytebrew.ai
            </a>{' '}
            to exercise these rights.
          </p>
        </div>

        <div>
          <h2 className="text-lg font-semibold text-brand-light">7. Cookies & Local Storage</h2>
          <ul className="mt-2 list-disc list-inside space-y-1 text-brand-shade3">
            <li>We use localStorage for authentication tokens</li>
            <li>No third-party tracking cookies</li>
            <li>No advertising cookies</li>
            <li>Optional: analytics via Plausible (privacy-friendly, no cookies)</li>
          </ul>
        </div>

        <div>
          <h2 className="text-lg font-semibold text-brand-light">8. Security</h2>
          <ul className="mt-2 list-disc list-inside space-y-1 text-brand-shade3">
            <li>All data transmitted over HTTPS/TLS</li>
            <li>Passwords hashed with bcrypt</li>
            <li>API tokens hashed with SHA-256</li>
            <li>Regular security reviews</li>
          </ul>
        </div>

        <div>
          <h2 className="text-lg font-semibold text-brand-light">9. Changes to This Policy</h2>
          <p className="mt-2">
            We may update this policy from time to time. Changes will be posted on this page with
            an updated &quot;Last modified&quot; date. Continued use after changes constitutes
            acceptance.
          </p>
        </div>

        <div>
          <h2 className="text-lg font-semibold text-brand-light">10. Contact</h2>
          <p className="mt-2">
            For privacy-related questions:{' '}
            <a
              href="mailto:privacy@bytebrew.ai"
              className="text-brand-accent hover:text-brand-accent-hover"
            >
              privacy@bytebrew.ai
            </a>
          </p>
          <p className="mt-2">
            For general support:{' '}
            <a
              href="mailto:support@bytebrew.ai"
              className="text-brand-accent hover:text-brand-accent-hover"
            >
              support@bytebrew.ai
            </a>
          </p>
        </div>
      </div>
    </div>
  );
}
