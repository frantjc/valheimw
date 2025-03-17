import type { MetaFunction } from "@remix-run/node";

export const meta: MetaFunction = () => {
  return [
    { title: "Sindri" },
  ];
};

export default function Index() {
  return (
    <main>
      Hello there!
    </main>
  );
}
