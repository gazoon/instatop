use Mix.Config

config :tg_bot,
       bot_name: "InstaRate",
       bot_username: "InstaRateBot",
       mongo_chats: [
         database: "local",
         host: "35.189.124.60",
         port: 27017,
       ],
       mongo_scheduler: [
         database: "local",
         host: "35.189.124.60",
         port: 27017,
       ],
       mongo_queue: [
         database: "local",
         host: "35.189.124.60",
         port: 27017,
         collection: "insta_queue",
         max_processing_time: 10000
       ],
       mongo_cache: [
         database: "local",
         host: "35.189.124.60",
         port: 27017,
       ]

config :nadia, token: "501332340:AAFqDbDgOx6K4GqfuV0dMlOMW5RzoEObtl4"
