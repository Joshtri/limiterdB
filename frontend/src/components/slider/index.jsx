import {Slider as BaseSlider} from '@base-ui/react/slider'
import {cn} from '@/lib/cn'

export function Slider({value, onValueChange, min = 0, max = 100, step = 1, disabled, className}) {
  return (
    <BaseSlider.Root
      value={value}
      onValueChange={onValueChange}
      min={min}
      max={max}
      step={step}
      disabled={disabled}
      className={cn('w-full', className)}
    >
      <BaseSlider.Control className="flex items-center py-2.5 touch-none select-none">
        <BaseSlider.Track className="relative h-1.5 w-full rounded-full bg-(--color-surface-3)">
          <BaseSlider.Indicator className="rounded-full bg-brand-500" />
          <BaseSlider.Thumb
            className={cn(
              'size-4 rounded-full border border-(--color-border-strong) bg-(--color-fg)',
              'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-500',
              'data-dragging:scale-110 transition-transform'
            )}
          />
        </BaseSlider.Track>
      </BaseSlider.Control>
    </BaseSlider.Root>
  )
}
