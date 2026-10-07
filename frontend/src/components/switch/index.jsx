import {Switch as BaseSwitch} from '@base-ui/react/switch'
import {cn} from '@/lib/cn'

export function Switch({checked, onCheckedChange, disabled, label, id}) {
  return (
    <div className="flex items-center gap-2">
      <BaseSwitch.Root
        id={id}
        checked={checked}
        onCheckedChange={onCheckedChange}
        disabled={disabled}
        className={cn(
          'flex h-5 w-9 shrink-0 rounded-full border border-(--color-border-strong) bg-(--color-surface-2) p-0.5',
          'transition-colors duration-150 ease-[ease]',
          'data-checked:bg-brand-500 data-checked:border-brand-500',
          'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-500'
        )}
      >
        <BaseSwitch.Thumb
          className={cn(
            'size-3.5 rounded-full bg-(--color-fg)',
            'transition-[translate] duration-150 ease-[ease]',
            'data-checked:translate-x-4'
          )}
        />
      </BaseSwitch.Root>
      {label && (
        <label htmlFor={id} className="text-sm text-(--color-fg) cursor-pointer select-none">
          {label}
        </label>
      )}
    </div>
  )
}
