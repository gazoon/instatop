defmodule Voting.Router do
  @moduledoc false

  use Plug.Router
  alias Voting.Voters.Storages.Mongo, as: VotersStorage
  alias Voting.Girls.Storages.Mongo, as: GirlsStorage
  alias Voting.Girls.Girl
  alias Instagram.Clients.Http, as: InstagramClient
  alias Voting.EloRating


  plug :match
  plug :dispatch

  get "/" do
    x = VotersStorage.can_vote?("1", "1", "2")
    IO.inspect(x)
#    x = VotersStorage.try_vote("1", "3", "2")
#    IO.inspect(x)
    IO.inspect Application.get_env(:voting, :foo)
    x = GirlsStorage.get_girl("1")
    IO.inspect(x)
    x = %{x | rating: x.rating + 10}
    x = %{x | wins: x.wins + 1}
    x = %{x | loses: x.loses + 1}
    x = %{x | matches: x.matches + 2}
    GirlsStorage.update_girl(x)
    g = %Girl{username: "32", rating: 222, photo: "llll", added_at: 2222}
    x = GirlsStorage.add_girl(g)
    IO.inspect(x)
    #    x = InstagramClient.get_media_owner("Bb1kN5llll9FwcF")
    #    IO.inspect(x)
    #    x = InstagramClient.is_photo?("BbJ6VOLFOAT")
    # TODO: cover with tests
    IO.inspect(EloRating.recalculate(0, 1000))
    IO.inspect(EloRating.recalculate(100, 0))
    conn
    |> send_resp(200, "Plug!")
  end

end
