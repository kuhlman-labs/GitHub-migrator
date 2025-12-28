import { describe, it, expect } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { StepIndicator } from './StepIndicator';

const stepTitles = ['Welcome', 'Source', 'Destination', 'Database', 'Review'];

describe('StepIndicator', () => {
  it('renders all step indicators', () => {
    render(
      <StepIndicator
        currentStep={1}
        totalSteps={5}
        stepTitles={stepTitles}
      />
    );

    stepTitles.forEach((title) => {
      expect(screen.getByText(title)).toBeInTheDocument();
    });
  });

  it('highlights current step', () => {
    render(
      <StepIndicator
        currentStep={3}
        totalSteps={5}
        stepTitles={stepTitles}
      />
    );

    // Current step title should be bold
    const destTitle = screen.getByText('Destination');
    expect(destTitle).toHaveStyle('font-weight: bold');
  });

  it('shows checkmarks for completed steps', () => {
    const { container } = render(
      <StepIndicator
        currentStep={3}
        totalSteps={5}
        stepTitles={stepTitles}
      />
    );

    // Steps 1 and 2 should have checkmarks (SVGs)
    const stepCircles = container.querySelectorAll('.w-8.h-8.rounded-full');
    
    // First two steps (completed) should have SVG checkmarks
    expect(stepCircles[0].querySelector('svg')).toBeInTheDocument();
    expect(stepCircles[1].querySelector('svg')).toBeInTheDocument();
    
    // Current step (3) should show the number
    expect(stepCircles[2].textContent).toBe('3');
  });

  it('shows step numbers for current and upcoming steps', () => {
    const { container } = render(
      <StepIndicator
        currentStep={2}
        totalSteps={5}
        stepTitles={stepTitles}
      />
    );

    const stepCircles = container.querySelectorAll('.w-8.h-8.rounded-full');
    
    // Step 1 (completed) should have checkmark
    expect(stepCircles[0].querySelector('svg')).toBeInTheDocument();
    
    // Step 2 (current) should show '2'
    expect(stepCircles[1].textContent).toBe('2');
    
    // Steps 3, 4, 5 (upcoming) should show their numbers
    expect(stepCircles[2].textContent).toBe('3');
    expect(stepCircles[3].textContent).toBe('4');
    expect(stepCircles[4].textContent).toBe('5');
  });

  it('renders single step correctly', () => {
    render(
      <StepIndicator
        currentStep={1}
        totalSteps={1}
        stepTitles={['Only Step']}
      />
    );

    expect(screen.getByText('Only Step')).toBeInTheDocument();
    expect(screen.getByText('1')).toBeInTheDocument();
  });

  it('handles first step correctly', () => {
    const { container } = render(
      <StepIndicator
        currentStep={1}
        totalSteps={5}
        stepTitles={stepTitles}
      />
    );

    const stepCircles = container.querySelectorAll('.w-8.h-8.rounded-full');
    
    // First step should show '1' (not checkmark since it's current)
    expect(stepCircles[0].textContent).toBe('1');
    
    // All other steps should show their numbers
    expect(stepCircles[1].textContent).toBe('2');
    expect(stepCircles[2].textContent).toBe('3');
  });

  it('handles last step correctly', () => {
    const { container } = render(
      <StepIndicator
        currentStep={5}
        totalSteps={5}
        stepTitles={stepTitles}
      />
    );

    const stepCircles = container.querySelectorAll('.w-8.h-8.rounded-full');
    
    // All previous steps should have checkmarks
    for (let i = 0; i < 4; i++) {
      expect(stepCircles[i].querySelector('svg')).toBeInTheDocument();
    }
    
    // Last step (current) should show '5'
    expect(stepCircles[4].textContent).toBe('5');
  });
});

