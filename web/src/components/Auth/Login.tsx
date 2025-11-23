import { useAuth } from '../../contexts/AuthContext';
import { Heading, Button, Flash } from '@primer/react';
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

          {/* Title */}
          <Heading as="h1" className="text-2xl text-center mb-2">
            GitHub Migration Server
          </Heading>
          <p className="text-gray-600 text-center mb-8">
            Sign in to continue
          </p>

          {/* Authorization Requirements */}
          {hasRules && (
            <Flash variant="default" className="mb-4">
              <p className="text-sm font-semibold mb-2">Access Requirements</p>
              <ul className="text-xs text-gray-600 pl-4 space-y-1">
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

          {/* Info Text */}
          <p className="mt-6 text-xs text-gray-500 text-center">
            You will be redirected to GitHub to authenticate
          </p>
        </div>
      </div>
    </div>
  );
}

