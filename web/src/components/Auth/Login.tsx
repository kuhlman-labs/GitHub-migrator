import { useState, useMemo } from 'react';
import { useAuth, type AuthSource } from '../../contexts/AuthContext';
import { Button, Heading, Text, FormControl, RadioGroup, Radio } from '@primer/react';
import { MarkGithubIcon } from '@primer/octicons-react';

export function Login() {
  const { authConfig, authSources, login } = useAuth();
  
  // Auto-select if only one source, otherwise start with null
  const defaultSourceId = useMemo(() => {
    return authSources.length === 1 ? authSources[0].id : null;
  }, [authSources]);
  
  const [selectedSourceId, setSelectedSourceId] = useState<number | null>(null);
  
  // Use the computed default if no selection has been made
  const effectiveSourceId = selectedSourceId ?? defaultSourceId;

  const hasRules = authConfig?.authorization_rules;
  const requiresOrg = hasRules?.requires_org_membership;
  const requiresTeam = hasRules?.requires_team_membership;
  const requiresEnterpriseAdmin = hasRules?.requires_enterprise_admin;
  const requiresEnterpriseMembership = hasRules?.requires_enterprise_membership;

  const hasMultipleSources = authSources.length > 1;
  const hasSources = authSources.length > 0;

  const handleLogin = () => {
    if (effectiveSourceId) {
      login(effectiveSourceId);
    } else {
      login();
    }
  };

  const getSourceTypeLabel = (type: AuthSource['type']) => {
    switch (type) {
      case 'github':
        return 'GitHub';
      case 'azuredevops':
        return 'Azure DevOps';
      default:
        return type;
    }
  };

  const getLoginButtonText = () => {
    if (effectiveSourceId && authSources.length > 0) {
      const source = authSources.find(s => s.id === effectiveSourceId);
      if (source?.type === 'azuredevops') {
        return 'Sign in with Microsoft';
      }
    }
    return 'Sign in with GitHub';
  };

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

          {/* Source Selector - Only show if multiple sources with OAuth */}
          {hasMultipleSources && (
            <div className="mb-6">
              <FormControl>
                <FormControl.Label>Select your source system</FormControl.Label>
                <RadioGroup
                  name="source"
                  onChange={(value) => setSelectedSourceId(value ? parseInt(value, 10) : null)}
                >
                  {authSources.map((source) => (
                    <FormControl key={source.id}>
                      <Radio 
                        value={String(source.id)} 
                        checked={effectiveSourceId === source.id}
                        onChange={() => setSelectedSourceId(source.id)}
                      />
                      <FormControl.Label>
                        {source.name}
                        <Text
                          as="span"
                          className="ml-2 text-xs"
                          style={{ color: 'var(--fgColor-muted)' }}
                        >
                          ({getSourceTypeLabel(source.type)})
                        </Text>
                      </FormControl.Label>
                    </FormControl>
                  ))}
                </RadioGroup>
              </FormControl>
              <Text
                as="p"
                className="mt-2 text-xs"
                style={{ color: 'var(--fgColor-muted)' }}
              >
                You'll authenticate against your selected source to verify repository access.
              </Text>
            </div>
          )}

          {/* Authorization Requirements */}
          {hasRules && !hasSources && (
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
                {requiresOrg && (
                  <li className="mb-1">
                    Enterprise member: {hasRules.required_orgs?.join(', ')}
                  </li>
                )}
                {requiresTeam && (
                  <li className="mb-1">
                    Team member: {hasRules.required_teams?.join(', ')}
                  </li>
                )}
                {requiresEnterpriseAdmin && (
                  <li className="mb-1">
                    Enterprise admin: {hasRules.enterprise}
                  </li>
                )}
                {requiresEnterpriseMembership && !requiresEnterpriseAdmin && (
                  <li className="mb-1">
                    Enterprise member: {hasRules.enterprise}
                  </li>
                )}
              </ul>
            </div>
          )}

          {/* Login Button */}
          <Button
            variant="primary"
            onClick={handleLogin}
            block
            leadingVisual={MarkGithubIcon}
            disabled={hasMultipleSources && !effectiveSourceId}
          >
            {getLoginButtonText()}
          </Button>

          {/* Info Text */}
          <Text
            as="p"
            className="mt-6 text-xs text-center"
            style={{ color: 'var(--fgColor-muted)' }}
          >
            {hasSources
              ? 'You will be redirected to authenticate with your source system'
              : 'You will be redirected to GitHub to authenticate'}
          </Text>
        </div>
      </div>
    </div>
  );
}
