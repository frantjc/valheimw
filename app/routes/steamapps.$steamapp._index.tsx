import type { MetaFunction } from "@remix-run/node";
import { getSteamapp } from "~/client";
import { LoaderFunctionArgs } from "@remix-run/node";
import { useLoaderData } from "@remix-run/react";

export const meta: MetaFunction = () => {
  return [
    { title: "Sindri" },
  ];
};

export function loader({ params }: LoaderFunctionArgs) {
  const { steamapp } = params;

  if (!steamapp) {
    throw new Response(null, {
      status: 404,
      statusText: "Steamapp not found",
    });
  }

  const steamappID = parseInt(steamapp)
  if (!steamappID) {
    throw new Response(null, {
      status: 404,
      statusText: `Invalid Steamapp ID ${steamapp}`,
    });
  }

  return getSteamapp(steamappID)
}

export default function Main() {
  const steamapp = useLoaderData<typeof loader>();

  return (
    <main>
      {JSON.stringify(steamapp)}
    </main>
  );
}
