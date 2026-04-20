import { useEffect } from 'react';

const REPO_URL = 'https://github.com/syntheticinc/bytebrew';

// GitHubStarButton renders the official GitHub "Star" button with a live
// star count. The underlying buttons.github.io script is idempotent —
// it replaces <a class="github-button"> nodes with an <iframe> once the
// script has loaded, so mounting the script multiple times is safe (it
// dedupes on the script src). We still guard against re-adding the tag
// so the DOM stays clean across route changes.
export function GitHubStarButton() {
  useEffect(() => {
    const scriptId = 'github-buttons-script';
    if (document.getElementById(scriptId)) return;

    const script = document.createElement('script');
    script.id = scriptId;
    script.src = 'https://buttons.github.io/buttons.js';
    script.async = true;
    script.defer = true;
    document.body.appendChild(script);
    // Note: we intentionally do NOT remove the script on unmount.
    // buttons.js attaches event listeners and replaces nodes; tearing it
    // down on every AdminLayout mount causes race conditions where the
    // button renders as a plain link.
  }, []);

  return (
    <a
      className="github-button"
      href={REPO_URL}
      data-color-scheme="no-preference: dark; light: dark; dark: dark;"
      data-show-count="true"
      data-size="large"
      aria-label="Star syntheticinc/bytebrew on GitHub"
      target="_blank"
      rel="noopener noreferrer"
    >
      Star
    </a>
  );
}

export default GitHubStarButton;
