import { useAuth } from '../../contexts/AuthContext';
import { Button, Heading, Text } from '@primer/react';
import { MarkGithubIcon } from '@primer/octicons-react';

export function Login() {
  const { authConfig, login } = useAuth();

  const hasRules = authConfig?.authorization_rules;
  const requiresOrg = hasRules?.requires_org_membership;
  const requiresTeam = hasRules?.requires_team_membership;
  const requiresEnterpriseAdmin = hasRules?.requires_enterprise_admin;

  return (
    <div
      className="min-h-screen flex items-center justify-center px-4"
      style={{ backgroundColor: 'var(--bgColor-inset)' }}
    >
      <div className="max-w-md w-full">
        <div
          className="rounded-lg border p-8 shadow-lg"
          style={{
            backgroundColor: 'var(--bgColor-default)',
            borderColor: 'var(--borderColor-default)',
          }}
        >
          {/* GitHub Logo */}
          <div className="flex justify-center mb-6">
            <MarkGithubIcon size={64} />
          </div>

          {/* Title */}
          <Heading
            as="h1"
            className="text-2xl font-semibold text-center mb-2"
          >
            GitHub Migrator
          </Heading>

          {/* Description */}
          <Text
            as="p"
            className="text-base text-center mb-6"
            style={{ color: 'var(--fgColor-muted)' }}
          >
            Sign in to continue
          </Text>

          {/* Authorization Requirements */}
          {hasRules && (
            <div
              className="mb-6 p-4 rounded-lg border"
              style={{
                backgroundColor: 'var(--bgColor-neutral-muted)',
                borderColor: 'var(--borderColor-muted)',
                opacity: 0.85,
              }}
            >
              <Text
                as="p"
                className="text-sm font-semibold mb-2"
                style={{ color: 'var(--fgColor-default)' }}
              >
                Access Requirements
              </Text>
              <ul
                className="text-xs pl-4 m-0 list-disc"
                style={{ color: 'var(--fgColor-default)' }}
              >
                {/* Enterprise requirement - shown first and prominently */}
                {hasRules.enterprise && (
                  <li className="mb-1">
                    {requiresEnterpriseAdmin 
                      ? `Enterprise admin of: ${hasRules.enterprise}`
                      : `Member of enterprise: ${hasRules.enterprise}`}
                  </li>
                )}
                {requiresOrg && (
                  <li className="mb-1">
                    Organization member: {hasRules.required_orgs?.join(', ')}
                  </li>
                )}
                {requiresTeam && (
                  <li className="mb-1">
                    Team member: {hasRules.required_teams?.join(', ')}
                  </li>
                )}
              </ul>
            </div>
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
          <Text
            as="p"
            className="mt-6 text-xs text-center"
            style={{ color: 'var(--fgColor-muted)' }}
          >
            You will be redirected to GitHub to authenticate
          </Text>
        </div>
      </div>
    </div>
  );
}
