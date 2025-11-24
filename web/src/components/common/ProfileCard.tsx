interface ProfileCardProps {
  title: string;
  children: React.ReactNode;
}

export function ProfileCard({ title, children }: ProfileCardProps) {
  return (
    <div 
      className="rounded-lg shadow-sm p-6"
      style={{ backgroundColor: 'var(--bgColor-default)' }}
    >
      <h3 className="text-lg font-medium mb-4" style={{ color: 'var(--fgColor-default)' }}>{title}</h3>
      <div className="space-y-3">{children}</div>
    </div>
  );
}

