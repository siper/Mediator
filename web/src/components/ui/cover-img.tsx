import { useState } from "react";

interface CoverImgProps {
  src: string;
  alt?: string;
  className?: string;
  noCover?: React.ReactNode;
}

export function CoverImg({ src, alt = "", className, noCover }: CoverImgProps) {
  const [error, setError] = useState(false);
  if (error || !src) {
    return noCover ?? null;
  }
  return (
    <img
      src={src}
      alt={alt}
      className={className}
      onError={() => setError(true)}
    />
  );
}
