interface ProfileCardProps {
  title: string;
  children: React.ReactNode;
}

export function ProfileCard({ title, children }: ProfileCardProps) {
  return (
    <div className="bg-white rounded-lg shadow-sm p-6">
      <h3 className="text-lg font-medium text-gray-900 mb-4">{title}</h3>
      <div className="space-y-3">{children}</div>
    </div>
  );
}

