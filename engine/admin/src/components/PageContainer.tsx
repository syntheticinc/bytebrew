import type { ReactNode } from 'react';

// PageContainer — canonical outer wrapper for every admin page.
//
// The <main> in Layout already applies p-6 and is the scroll parent. This
// component only handles horizontal width: a fixed max width and auto-centering
// so content has a consistent left offset from the sidebar regardless of which
// page is active.
//
// Width choice: 1200px matches the pre-existing reference pages (OverviewPage,
// SchemasPage) which were deemed "correct" — all other pages were drifting.
//
// Variants:
//   - default:       max-w-[1200px] mx-auto  — the standard page chrome
//   - wide=true:     no max-width constraint  — for pages whose content is a
//                    canvas or full-bleed editor and needs every horizontal
//                    pixel (SchemaDetailPage flow graph, AgentDrillInPage,
//                    WidgetConfigPage split layout). Such pages MUST supply a
//                    justification comment at the <PageContainer wide> site.
//   - narrow=true:   max-w-3xl mx-auto  — for narrow form-only pages
//                    (SettingsPage, ConfigPage) where 1200px of whitespace
//                    would look empty.
//
// `wide` and `narrow` are mutually exclusive; narrow wins if both are set.
export interface PageContainerProps {
  children: ReactNode;
  /** Drop the max-width cap. Use for canvases and full-bleed editors only. */
  wide?: boolean;
  /** Narrow 3xl container. Use for form-only pages like Settings/Config. */
  narrow?: boolean;
  /** Extra classes merged onto the wrapper. */
  className?: string;
}

export default function PageContainer({ children, wide, narrow, className }: PageContainerProps) {
  const widthClass = narrow
    ? 'max-w-3xl mx-auto'
    : wide
      ? 'w-full'
      : 'max-w-[1200px] mx-auto';
  const classes = `${widthClass}${className ? ` ${className}` : ''}`;
  return <div className={classes}>{children}</div>;
}
