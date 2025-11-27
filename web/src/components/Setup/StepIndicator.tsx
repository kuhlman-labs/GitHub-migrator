import { Text } from '@primer/react';
import { CheckIcon } from '@primer/octicons-react';

interface StepIndicatorProps {
  currentStep: number;
  totalSteps: number;
  stepTitles: string[];
}

export function StepIndicator({ currentStep, totalSteps, stepTitles }: StepIndicatorProps) {
  return (
    <div
      className="flex justify-center items-center gap-4 mb-6 pb-6 border-b"
      style={{ borderColor: 'var(--borderColor-default)' }}
    >
      {Array.from({ length: totalSteps }, (_, i) => i + 1).map((step) => (
        <div
          key={step}
          className="flex flex-col items-center flex-1"
          style={{ maxWidth: '120px' }}
        >
          <div
            className="w-8 h-8 rounded-full flex items-center justify-center mb-2 font-bold"
            style={{
              backgroundColor: step < currentStep 
                ? 'var(--bgColor-success-emphasis)' 
                : step === currentStep 
                ? 'var(--bgColor-accent-emphasis)' 
                : 'var(--bgColor-muted)',
              color: step <= currentStep ? 'white' : 'var(--fgColor-muted)',
            }}
          >
            {step < currentStep ? <CheckIcon size={16} /> : step}
          </div>
          <Text
            className="text-xs text-center"
            style={{
              color: step === currentStep ? 'var(--fgColor-default)' : 'var(--fgColor-muted)',
              fontWeight: step === currentStep ? 'bold' : 'normal',
            }}
          >
            {stepTitles[step - 1]}
          </Text>
        </div>
      ))}
    </div>
  );
}
