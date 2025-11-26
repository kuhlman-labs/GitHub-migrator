import { useAuth } from '../../contexts/AuthContext';
import { Button, Flash } from '@primer/react';
import { MarkGithubIcon } from '@primer/octicons-react';

export function Login() {
  const { authConfig, login } = useAuth();

  const hasRules = authConfig?.authorization_rules;
  const requiresOrg = hasRules?.requires_org_membership;
  const requiresTeam = hasRules?.requires_team_membership;
  const requiresEnterpriseAdmin = hasRules?.requires_enterprise_admin;
  const requiresEnterpriseMembership = hasRules?.requires_enterprise_membership;

  return (
    <div className="min-h-screen bg-gh-canvas-inset flex items-center justify-center px-4">
      <div className="max-w-md w-full">
        <div className="bg-white border border-gh-border-default rounded-lg p-8 shadow-lg">
          {/* GitHub Logo */}
          <div className="flex justify-center mb-6">
            <MarkGithubIcon size={64} />
          </div>

          {/* Title - Using standard h1 with Primer-aligned classes (semibold for headings) */}
          <h1 className="text-2xl font-semibold text-center mb-2">
            GitHub Migration Server
          </h1>
          {/* Description - Using standard p with normal weight (no font-weight class = 400) */}
          <p className="text-base text-center mb-8" style={{ color: 'var(--fgColor-muted)' }}>
            Sign in to continue
          </p>

          {/* Authorization Requirements */}
          {hasRules && (
            <Flash variant="default" className="mb-4">
              {/* Use semibold only for the label/heading */}
              <p className="text-sm font-semibold mb-2">Access Requirements</p>
              {/* List items use normal weight (400) - no font-weight class */}
              <ul className="text-xs pl-4 space-y-1 list-disc" style={{ color: 'var(--fgColor-muted)' }}>
                {requiresOrg && (
                  <li>
                    Organization member: {hasRules.required_orgs?.join(', ')}
                  </li>
                )}
                {requiresTeam && (
                  <li>
                    Team member: {hasRules.required_teams?.join(', ')}
                  </li>
                )}
                {requiresEnterpriseAdmin && (
                  <li>
                    Enterprise admin: {hasRules.enterprise}
                  </li>
                )}
                {requiresEnterpriseMembership && !requiresEnterpriseAdmin && (
                  <li>
                    Enterprise member: {hasRules.enterprise}
                  </li>
                )}
              </ul>
            </Flash>
          )}

          {/* Login Button */}
          <Button
            variant="primary"
            onClick={login}
            block
            leadingVisual={MarkGithubIcon}
          >
            Sign in with GitHub
          </Button>

          {/* Info Text - caption size with normal weight */}
          <p className="mt-6 text-xs text-center" style={{ color: 'var(--fgColor-muted)' }}>
            You will be redirected to GitHub to authenticate
          </p>
        </div>
      </div>
    </div>
  );
}

