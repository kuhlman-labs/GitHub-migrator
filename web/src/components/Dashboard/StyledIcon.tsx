import { ReactElement, cloneElement } from 'react';

interface StyledIconProps {
  icon: ReactElement;
  color: string;
  className?: string;
}

export function StyledIcon({ icon, color, className }: StyledIconProps) {
  return (
    <span style={{ color }} className={className}>
      {cloneElement(icon)}
    </span>
  );
}
