import { Label } from '@primer/react';

interface BadgeProps {
  children: React.ReactNode;
  color?: 'blue' | 'green' | 'yellow' | 'red' | 'purple' | 'gray' | 'orange' | 'pink' | 'indigo' | 'teal';
}

export function Badge({ children, color = 'gray' }: BadgeProps) {
  // Map color to Primer Label variants
  const variantMap: Record<string, 'default' | 'primary' | 'secondary' | 'accent' | 'success' | 'attention' | 'severe' | 'danger' | 'done' | 'sponsors'> = {
    blue: 'accent',
    green: 'success',
    yellow: 'attention',
    red: 'danger',
    purple: 'sponsors',
    gray: 'default',
    orange: 'attention',
    pink: 'sponsors',
    indigo: 'accent',
    teal: 'accent',
  };
  
  return (
    <Label variant={variantMap[color]}>
      {children}
    </Label>
  );
}

