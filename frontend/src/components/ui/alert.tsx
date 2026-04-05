import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "../../lib/utils"

const alertVariants = cva("relative w-full rounded-[1.5rem] border px-5 py-4", {
  variants: {
    variant: {
      default: "border-border/70 bg-background/70 text-foreground",
      warning: "border-warning/20 bg-warning/10 text-warning",
      error: "border-destructive/20 bg-destructive/10 text-destructive",
      success: "border-success/20 bg-success/10 text-success",
    },
  },
  defaultVariants: {
    variant: "default",
  },
})

const Alert = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement> & VariantProps<typeof alertVariants>
>(({ className, variant, ...props }, ref) => (
  <div ref={ref} role="alert" className={cn(alertVariants({ variant }), className)} {...props} />
))
Alert.displayName = "Alert"

const AlertTitle = React.forwardRef<HTMLHeadingElement, React.HTMLAttributes<HTMLHeadingElement>>(
  ({ className, ...props }, ref) => (
    <h5 ref={ref} className={cn("mb-1 font-display text-base font-semibold tracking-[-0.04em]", className)} {...props} />
  ),
)
AlertTitle.displayName = "AlertTitle"

const AlertDescription = React.forwardRef<HTMLParagraphElement, React.HTMLAttributes<HTMLParagraphElement>>(
  ({ className, ...props }, ref) => <div ref={ref} className={cn("text-sm leading-7", className)} {...props} />,
)
AlertDescription.displayName = "AlertDescription"

export { Alert, AlertTitle, AlertDescription }
