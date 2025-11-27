import { Heading } from '@primer/react';
import { MarkGithubIcon } from '@primer/octicons-react';
import { SetupWizard } from './SetupWizard';

export function Setup() {
  return (
    <div
      className="min-h-screen py-8"
      style={{ backgroundColor: 'var(--bgColor-inset)' }}
    >
      <div className="max-w-5xl mx-auto px-6">
        {/* Header */}
        <div
          className="text-center mb-8 pb-6 border-b"
          style={{ borderColor: 'var(--borderColor-default)' }}
        >
          <div className="flex justify-center mb-4">
            <MarkGithubIcon size={64} />
          </div>
          <Heading as="h1" className="text-4xl mb-3">
            GitHub Migrator
          </Heading>
          <Heading
            as="h2"
            className="text-2xl font-normal"
            style={{ color: 'var(--fgColor-muted)' }}
          >
            Setup
          </Heading>
        </div>

        {/* Wizard */}
        <div
          className="rounded-lg border p-6"
          style={{
            backgroundColor: 'var(--bgColor-default)',
            borderColor: 'var(--borderColor-default)',
          }}
        >
          <SetupWizard />
        </div>
      </div>
    </div>
  );
}
