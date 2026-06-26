import { cn } from '@/lib/utils'

type BrandLogoProps = {
  size?: 'sm' | 'lg'
  className?: string
}

const shellSizeClass: Record<NonNullable<BrandLogoProps['size']>, string> = {
  sm: 'size-[34px] rounded-[9px]',
  lg: 'size-11 rounded-xl',
}

const glyphSize: Record<NonNullable<BrandLogoProps['size']>, number> = {
  sm: 18,
  lg: 20,
}

export default function BrandLogo({ size = 'sm', className }: BrandLogoProps) {
  return (
    <div
      aria-hidden="true"
      className={cn(
        'grid shrink-0 place-items-center border border-[hsl(240_8%_18%)] bg-linear-to-br from-[hsl(220_8%_10%)] to-[hsl(240_9%_4%)] shadow-[inset_0_1px_0_hsl(0_0%_100%/0.06),0_0_22px_hsl(186_31%_50%/0.15)]',
        shellSizeClass[size],
        className,
      )}
    >
      <svg width={glyphSize[size]} height={glyphSize[size]} viewBox="0 0 24 24" fill="none">
        <circle cx="10.8" cy="10.8" r="7" stroke="hsl(var(--foreground))" strokeWidth="1.45" opacity="0.18" />
        <path d="M16 16l4.5 4.5" stroke="hsl(var(--foreground))" strokeWidth="1.7" strokeLinecap="round" opacity="0.55" />
        <path d="M4.8 14.8l3.3-4 3.1 2.8 5.1-6.1" stroke="hsl(var(--accent))" strokeWidth="2.1" strokeLinecap="round" strokeLinejoin="round" />
        <circle cx="8.1" cy="10.8" r="1.35" fill="hsl(186 28% 64%)" />
        <circle cx="16.3" cy="7.5" r="1.35" fill="hsl(186 28% 64%)" />
      </svg>
    </div>
  )
}
