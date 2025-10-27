import { useAuth } from '../../contexts/AuthContext';

export function Login() {
  const { authConfig, login } = useAuth();

  const hasRules = authConfig?.authorization_rules;
  const requiresOrg = hasRules?.requires_org_membership;
  const requiresTeam = hasRules?.requires_team_membership;
  const requiresEnterpriseAdmin = hasRules?.requires_enterprise_admin;

  return (
    <div className="min-h-screen bg-gh-canvas-inset flex items-center justify-center px-4">
      <div className="max-w-md w-full">
        <div className="bg-white border border-gh-border-default rounded-lg p-8 shadow-lg">
          {/* GitHub Logo */}
          <div className="flex justify-center mb-6">
            <svg
              className="h-16 w-16 text-gray-800"
              fill="currentColor"
              viewBox="0 0 16 16"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z" />
            </svg>
          </div>

          {/* Title */}
          <h1 className="text-2xl font-semibold text-gray-900 text-center mb-2">
            GitHub Migration Server
          </h1>
          <p className="text-gray-600 text-center mb-8">
            Sign in to continue
          </p>

          {/* Authorization Requirements */}
          {hasRules && (
            <div className="mb-6 p-4 bg-gray-50 border border-gray-200 rounded-md">
              <h3 className="text-sm font-semibold text-gray-900 mb-2">
                Access Requirements
              </h3>
              <ul className="text-sm text-gray-600 space-y-1">
                {requiresOrg && (
                  <li>
                    • Organization member: {hasRules.required_orgs?.join(', ')}
                  </li>
                )}
                {requiresTeam && (
                  <li>
                    • Team member: {hasRules.required_teams?.join(', ')}
                  </li>
                )}
                {requiresEnterpriseAdmin && (
                  <li>
                    • Enterprise admin: {hasRules.enterprise}
                  </li>
                )}
              </ul>
            </div>
          )}

          {/* Login Button */}
          <button
            onClick={login}
            className="w-full bg-gh-success-emphasis hover:bg-gh-success-emphasis/90 text-white font-medium py-3 px-4 rounded-md transition-colors flex items-center justify-center gap-2"
          >
            <svg
              className="w-5 h-5"
              fill="currentColor"
              viewBox="0 0 16 16"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z" />
            </svg>
            Sign in with GitHub
          </button>

          {/* Info Text */}
          <p className="mt-6 text-xs text-gray-500 text-center">
            You will be redirected to GitHub to authenticate
          </p>
        </div>
      </div>
    </div>
  );
}

