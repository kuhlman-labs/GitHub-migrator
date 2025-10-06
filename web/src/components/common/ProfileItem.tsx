interface ProfileItemProps {
  label: string;
  value: React.ReactNode;
}

export function ProfileItem({ label, value }: ProfileItemProps) {
  return (
    <div className="flex justify-between items-center">
      <span className="text-sm text-gray-600">{label}:</span>
      <span className="text-sm font-medium text-gray-900">{value}</span>
    </div>
  );
}

