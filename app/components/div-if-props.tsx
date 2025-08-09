export function DivIfProps({
  children,
  ...rest
}: React.DetailedHTMLProps<
  React.HTMLAttributes<HTMLDivElement>,
  HTMLDivElement
>) {
  if (Object.keys(rest).length) {
    return <div {...rest}>{children}</div>;
  }

  return <>{children}</>;
}
