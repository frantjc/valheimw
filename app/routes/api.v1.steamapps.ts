export function loader() {
  return new Response(
    JSON.stringify({
      steamapps: [
        {
          app_id: 896660,
          name: "Valheim Dedicated Server",
          branch: "public-test",
          icon_url: "https://cdn.cloudflare.steamstatic.com/steamcommunity/public/images/apps/896660/1aab0586723c8578c7990ced7d443568649d0df2.jpg",
          date_created: new Date(),
          locked: false
        },
        {
          app_id: 896660,
          name: "Valheim Dedicated Server",
          icon_url: "https://cdn.cloudflare.steamstatic.com/steamcommunity/public/images/apps/896660/1aab0586723c8578c7990ced7d443568649d0df2.jpg",
          date_created: new Date(),
          locked: false
        }
      ],
    }),
    {
      headers: {
        "Content-Type": "application/json",
      },
    },
  );
}
