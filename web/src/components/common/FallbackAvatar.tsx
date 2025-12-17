import { useState, useCallback } from 'react';
import { PersonIcon } from '@primer/octicons-react';

interface FallbackAvatarProps {
  src?: string;
  login?: string;
  size?: number;
  className?: string;
}

// Generate a consistent color from a string (for fallback background)
function stringToColor(str: string): string {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash);
  }
  
  // Generate a hue from 0-360, keeping saturation and lightness constant for readability
  const hue = Math.abs(hash) % 360;
  return `hsl(${hue}, 45%, 35%)`;
}

// Get initials from login (first 2 characters, uppercase)
function getInitials(login: string): string {
  return login.slice(0, 2).toUpperCase();
}

/**
 * Avatar component with graceful fallback for authenticated/inaccessible URLs.
 * Falls back to showing user initials on a colored background when the image fails to load.
 */
export function FallbackAvatar({ src, login = '', size = 24, className = '' }: FallbackAvatarProps) {
  const [imageError, setImageError] = useState(false);
  const [imageLoaded, setImageLoaded] = useState(false);

  const handleError = useCallback(() => {
    setImageError(true);
  }, []);

  const handleLoad = useCallback(() => {
    setImageLoaded(true);
  }, []);

  const showFallback = imageError || !src;
  const backgroundColor = login ? stringToColor(login) : 'var(--bgColor-neutral-muted)';
  const initials = login ? getInitials(login) : '';

  return (
    <div
      className={`inline-flex items-center justify-center rounded-full overflow-hidden flex-shrink-0 ${className}`}
      style={{
        width: size,
        height: size,
        backgroundColor: showFallback ? backgroundColor : 'transparent',
        fontSize: Math.max(10, size * 0.4),
        fontWeight: 600,
        color: '#ffffff',
      }}
    >
      {!showFallback && (
        <img
          src={src}
          alt={login || 'User avatar'}
          width={size}
          height={size}
          onError={handleError}
          onLoad={handleLoad}
          style={{
            display: imageLoaded ? 'block' : 'none',
            width: '100%',
            height: '100%',
            objectFit: 'cover',
          }}
        />
      )}
      {/* Show fallback while loading (if no error yet) or when error occurs */}
      {(showFallback || !imageLoaded) && (
        <span
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            width: '100%',
            height: '100%',
            backgroundColor,
          }}
        >
          {initials || <PersonIcon size={Math.max(12, size * 0.5)} />}
        </span>
      )}
    </div>
  );
}

