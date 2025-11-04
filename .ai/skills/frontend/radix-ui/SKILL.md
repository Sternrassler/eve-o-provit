# Frontend Skill: Radix UI Components

**Tech Stack:** Radix UI + shadcn/ui + Tailwind CSS 4

**Project:** eve-o-provit UI Components

---

## Architecture Patterns

### Unstyled Primitives
- **Radix UI:** Provides accessibility, behavior, and keyboard navigation
- **shadcn/ui:** Pre-styled Radix components with Tailwind CSS
- **Customization:** Modify `components/ui/` files to match design

### Controlled Components
- **State Management:** Parent component manages state
- **Event Handlers:** `onValueChange`, `onCheckedChange`, etc.
- **Type Safety:** TypeScript interfaces for all props

---

## Best Practices

1. **Accessibility First:** Radix components are ARIA-compliant by default
2. **Controlled Pattern:** Always provide `value` + `onValueChange` (not uncontrolled)
3. **Composition:** Combine primitives to build complex components
4. **Styling:** Use Tailwind utility classes, avoid inline styles
5. **Variants:** Use `class-variance-authority` for component variants
6. **Portal Usage:** Dialogs and Tooltips render in portal (avoid z-index issues)
7. **Form Integration:** Radix components work with React Hook Form

---

## Common Patterns

### 1. Select Component

```tsx
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

<Select value={value} onValueChange={setValue}>
  <SelectTrigger className="w-[200px]">
    <SelectValue placeholder="Choose option" />
  </SelectTrigger>
  <SelectContent>
    <SelectItem value="1">Option 1</SelectItem>
    <SelectItem value="2">Option 2</SelectItem>
  </SelectContent>
</Select>
```

### 2. Button with Variants

```tsx
import { Button } from "@/components/ui/button";

<Button variant="default">Primary</Button>
<Button variant="outline">Secondary</Button>
<Button variant="destructive">Delete</Button>
<Button variant="ghost" size="sm">Small Ghost</Button>
```

### 3. Toast Notifications

```tsx
import { toast } from "@/hooks/use-toast";

const handleRefresh = () => {
  toast({
    title: "Success",
    description: "Data refreshed successfully",
  });
  
  // Error toast
  toast({
    title: "Error",
    description: "Failed to refresh data",
    variant: "destructive",
  });
};
```

### 4. Dialog (Modal)

```tsx
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";

<Dialog>
  <DialogTrigger asChild>
    <Button>Open Dialog</Button>
  </DialogTrigger>
  <DialogContent>
    <DialogHeader>
      <DialogTitle>Confirm Action</DialogTitle>
    </DialogHeader>
    <p>Are you sure you want to proceed?</p>
  </DialogContent>
</Dialog>
```

---

## Anti-Patterns

❌ **Uncontrolled Components:** Don't use `defaultValue` without managing state

❌ **Missing `asChild`:** Use `asChild` prop when wrapping trigger elements

❌ **Hardcoded Styles:** Use Tailwind classes, not inline `style={{...}}`

❌ **Z-Index Wars:** Radix portals handle stacking automatically

❌ **Accessibility Bypass:** Don't remove ARIA attributes from Radix components

---

## Integration with Tailwind CSS

### Utility Classes
```tsx
<Button className="hover:bg-primary/90 transition-colors">
  Hover Effect
</Button>
```

### Conditional Styling
```tsx
import { cn } from "@/lib/utils"; // tailwind-merge + clsx

<div className={cn(
  "px-3 py-1 rounded",
  isActive ? "bg-blue-500" : "bg-gray-200"
)} />
```

---

## Component Variants

### Using `class-variance-authority`
```tsx
import { cva } from "class-variance-authority";

const buttonVariants = cva(
  "inline-flex items-center justify-center rounded-md", // base
  {
    variants: {
      variant: {
        default: "bg-primary text-white",
        outline: "border border-gray-300",
      },
      size: {
        sm: "h-9 px-3 text-sm",
        lg: "h-11 px-8 text-lg",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "sm",
    },
  }
);
```

---

## Performance Considerations

- **Portal Rendering:** Dialogs/Tooltips render in portal (no layout shifts)
- **Lazy Loading:** Import heavy components only when needed
- **Animation Performance:** Use Tailwind's `transition-*` utilities (GPU-accelerated)

---

## Security Guidelines

- **XSS Prevention:** Radix components escape user input by default
- **ARIA Attributes:** Never remove accessibility attributes manually

---

## Quick Reference

| Component | Usage |
|-----------|-------|
| Button | `<Button variant="outline">Text</Button>` |
| Select | `<Select value={val} onValueChange={setVal}>...</Select>` |
| Dialog | `<Dialog><DialogTrigger>...</DialogTrigger></Dialog>` |
| Toast | `toast({ title: "...", description: "..." })` |
| Tooltip | `<Tooltip><TooltipTrigger>...</TooltipTrigger></Tooltip>` |
| Checkbox | `<Checkbox checked={val} onCheckedChange={setVal} />` |
| Slider | `<Slider value={[val]} onValueChange={([v]) => setVal(v)} />` |
