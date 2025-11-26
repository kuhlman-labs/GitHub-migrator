interface ProfileItemProps {
  label: string;
  value: React.ReactNode;
}

export function ProfileItem({ label, value }: ProfileItemProps) {
  return (
    <div className="flex justify-between items-center">
      <span className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>{label}:</span>
      <span className="text-sm font-medium" style={{ color: 'var(--fgColor-default)' }}>{value}</span>
    </div>
  );
}

