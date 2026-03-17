export function TermsPage() {
  return (
    <div className="max-w-3xl mx-auto px-4 py-10">
      <h1 className="text-2xl font-bold text-white">Terms of Service</h1>
      <p className="mt-2 text-sm text-gray-400">Last updated: February 23, 2026</p>

      <div className="mt-8 space-y-8 text-sm text-gray-300 leading-relaxed">
        <div>
          <h2 className="text-lg font-semibold text-white">1. Introduction</h2>
          <p className="mt-2">
            Welcome to ByteBrew Cloud, a software-as-a-service platform that provides AI agent
            capabilities including LLM proxy, licensing, and team collaboration tools. These Terms of
            Service ("Terms") govern your access to and use of ByteBrew Cloud services ("Service").
          </p>
          <p className="mt-2">
            By accessing or using the Service, you agree to be bound by these Terms. If you do not
            agree to these Terms, you may not access or use the Service.
          </p>
        </div>

        <div>
          <h2 className="text-lg font-semibold text-white">2. Account</h2>
          <p className="mt-2">
            You must create an account to use the Service. You are responsible for maintaining the
            confidentiality of your account credentials and for all activity that occurs under your
            account.
          </p>
          <p className="mt-2">
            Each person may maintain only one account. You must be at least 18 years of age to
            create an account and use the Service. You agree to provide accurate and complete
            information when creating your account and to keep this information up to date.
          </p>
        </div>

        <div>
          <h2 className="text-lg font-semibold text-white">3. Subscription & Billing</h2>
          <p className="mt-2">
            The Service offers paid subscription plans billed through Stripe. Subscriptions
            automatically renew at the end of each billing period unless cancelled before the renewal
            date.
          </p>
          <p className="mt-2 font-medium text-white">
            All payments are non-refundable.
          </p>
          <p className="mt-2">
            If you cancel your subscription, you will retain access to paid features until the end of
            your current billing period.
          </p>
          <p className="mt-2">
            We reserve the right to change subscription prices with at least 30 days' prior notice.
            Continued use of the Service after a price change constitutes acceptance of the new
            pricing.
          </p>
        </div>

        <div>
          <h2 className="text-lg font-semibold text-white">4. Acceptable Use</h2>
          <p className="mt-2">You agree not to:</p>
          <ul className="mt-2 list-disc list-inside space-y-1 text-gray-400">
            <li>Abuse, overload, or interfere with the API or any part of the Service</li>
            <li>Reverse engineer, decompile, or disassemble the Service or its components</li>
            <li>Generate excessive or unreasonable load on the Service infrastructure</li>
            <li>Use the Service for any illegal, fraudulent, or harmful purpose</li>
            <li>Share your account credentials with third parties or allow unauthorized access</li>
            <li>Attempt to circumvent usage limits, licensing restrictions, or security measures</li>
          </ul>
        </div>

        <div>
          <h2 className="text-lg font-semibold text-white">5. Intellectual Property</h2>
          <p className="mt-2">
            The Service, including all software, documentation, and associated intellectual property,
            is owned by ByteBrew Cloud and its licensors. Your subscription grants you a limited,
            non-exclusive, non-transferable license to use the Service in accordance with these
            Terms.
          </p>
          <p className="mt-2">
            You retain all rights to the content you create, submit, or generate through the
            Service. We claim no ownership over your content.
          </p>
        </div>

        <div>
          <h2 className="text-lg font-semibold text-white">6. Data & Privacy</h2>
          <p className="mt-2">
            We process your data solely to provide and improve the Service. LLM queries are proxied
            through our infrastructure to third-party model providers. We do not store the content of
            your prompts or model responses beyond what is necessary for real-time processing.
          </p>
          <p className="mt-2">
            We may collect usage metadata (such as request counts, feature usage, and performance
            metrics) to operate and improve the Service. For full details on how we handle your data,
            please refer to our Privacy Policy (to be published).
          </p>
        </div>

        <div>
          <h2 className="text-lg font-semibold text-white">7. Limitation of Liability</h2>
          <p className="mt-2">
            The Service is provided on an "AS IS" and "AS AVAILABLE" basis without warranties of any
            kind, whether express or implied, including but not limited to implied warranties of
            merchantability, fitness for a particular purpose, and non-infringement.
          </p>
          <p className="mt-2">
            To the maximum extent permitted by applicable law, ByteBrew Cloud shall not be liable for
            any indirect, incidental, special, consequential, or punitive damages. Our total
            liability for any claim arising out of or relating to these Terms or the Service shall
            not exceed the amount you paid to us in the 12 months preceding the claim.
          </p>
        </div>

        <div>
          <h2 className="text-lg font-semibold text-white">8. Termination</h2>
          <p className="mt-2">
            We may suspend or terminate your access to the Service at any time if you violate these
            Terms or engage in conduct that we reasonably determine to be harmful to the Service, its
            users, or third parties.
          </p>
          <p className="mt-2">
            You may delete your account at any time through the Settings page. Upon account deletion,
            your active subscription will be cancelled immediately and no refund will be issued for
            the remaining billing period. All associated data will be permanently removed.
          </p>
        </div>

        <div>
          <h2 className="text-lg font-semibold text-white">9. Changes to Terms</h2>
          <p className="mt-2">
            We may update these Terms from time to time. We will notify you of material changes via
            email or through a notice on the Service website. Your continued use of the Service after
            such changes constitutes acceptance of the updated Terms.
          </p>
        </div>

        <div>
          <h2 className="text-lg font-semibold text-white">10. Contact</h2>
          <p className="mt-2">
            If you have questions about these Terms, please contact us at{' '}
            <a
              href="mailto:support@bytebrewcloud.dev"
              className="text-indigo-400 hover:text-indigo-300"
            >
              support@bytebrewcloud.dev
            </a>
            .
          </p>
        </div>
      </div>
    </div>
  );
}
