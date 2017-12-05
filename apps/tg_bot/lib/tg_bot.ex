defmodule TGBot do

  require Logger
  alias TGBot.Messages.Text, as: TextMessage
  alias TGBot.Messages.Callback, as: Callback
  alias Voting.Girls.Girl
  alias TGBot.Messenger

  @start_cmd "start"
  @add_girl_cmd "addgirl"
  @get_top_cmd "showtop"
  @get_girl_info_cmd "girlinfo"
  @help_cmd "help"

  @vote_callback "vt"
  @get_top_callback "top"

  @usernames_separator "|"

  @top_page_size 10

  @spec on_message(map()) :: any
  def on_message(message_container) do
    message_type = message_container.type
    message_data = message_container.data
    case message_type do
      :text ->
        message = TextMessage.from_data(message_data)
        on_text_message(message)
      :callback ->
        message = Callback.from_data(message_data)
        on_callback(message)
      _ -> Logger.error("Unsupported message type: #{message_type}")
    end
  end

  @spec on_text_message(TextMessage.t) :: any
  defp on_text_message(message) do
    if TextMessage.appeal_to_bot?(message) || !message.is_group_chat do
      process_text_message(message)
    else
      Logger.info("Skip message #{inspect message} it's not an appeal")
    end
  end

  @spec process_text_message(TextMessage.t) :: any
  defp process_text_message(message) do
    Logger.info("Process text message #{inspect message}")
    commands = [
      {@start_cmd, &handle_start_cmd/1},
      {@add_girl_cmd, &handle_add_girl_cmd/1},
      {@get_top_cmd, &handle_get_top_cmd/1},
      {@get_girl_info_cmd, &handle_get_girl_info_cmd/1},
      {@help_cmd, &handle_help_cmd/1},
    ]
    message_text = message.text_lowercase
    command = commands
              |> Enum.find(fn ({cmd_name, _}) -> String.contains?(message_text, cmd_name) end)
    case command do
      {command_name, handler} -> Logger.info("Handle #{command_name} command")
                                 handler.(message)
      nil -> Logger.info("Message #{message_text} doesn't contain commands")
    end
  end

  @spec build_voter_id(integer) :: String.t
  defp build_voter_id(user_id) do
    "tg_user:" <> Integer.to_string(user_id)
  end

  @spec build_voters_group_id(integer) :: String.t
  defp build_voters_group_id(chat_id) do
    "tg_chat:" <> Integer.to_string(chat_id)
  end

  @spec handle_start_cmd(TextMessage.t) :: any
  defp handle_start_cmd(message) do
    send_next_girls_pair(message.chat_id)
  end

  @spec send_next_girls_pair(integer) :: any
  defp send_next_girls_pair(chat_id) do
    voters_group_id = build_voters_group_id(chat_id)
    {girl_one, girl_two} = Voting.get_next_pair(voters_group_id)
    girl_one_url = Girl.get_profile_url(girl_one)
    girl_two_url = Girl.get_profile_url(girl_two)

    match_photo = Pictures.concatenate(girl_one.photo, girl_two.photo)

    keyboard = [
      [
        %{
          text: "Left",
          payload: Callback.build_payload(
            @vote_callback,
            girl_one.username <> @usernames_separator <> girl_two.username
          )
        },
        %{
          text: "Right",
          payload: Callback.build_payload(
            @vote_callback,
            girl_two.username <> @usernames_separator <> girl_one.username
          )
        },
      ]
    ]
    caption_text = "#{girl_one_url} vs #{girl_two_url}"
    try do
      Messenger.send_photo(chat_id, match_photo, keyboard: keyboard, caption: caption_text)
    after
      File.rm!(match_photo)
    end
  end

  @spec handle_add_girl_cmd(TextMessage.t) :: any
  defp handle_add_girl_cmd(message) do
    photo_link = TextMessage.get_command_arg(message)
    case Voting.add_girl(photo_link) do
      {:ok, girl} ->
        profile_url = Girl.get_profile_url(girl)
        text = "Girl [#{girl.username}](#{profile_url}) has been successfully added!"
        Messenger.send_markdown(message.chat_id, text)
      {:error, error_msg} -> Messenger.send_text(message.chat_id, error_msg)
    end
  end
  #  @spec handle_get_top_cmd(TextMessage.t) :: any
  #  defp handle_get_top_cmd(message) do
  #    Voting.get_top(@top_page_size)
  #    |> Enum.with_index(1)
  #    |> Enum.each(
  #         fn ({girl, i}) ->
  #           Messenger.send_text(message.chat_id, "#{i}th place:")
  #           Messenger.send_photo(message.chat_id, girl.photo, caption: Girl.get_profile_url(girl))
  #         end
  #       )
  #  end

  #  @spec handle_get_top_cmd(TextMessage.t) :: any
  #  defp handle_get_top_cmd(message) do
  #    0..5
  #    |> Enum.each(
  #         fn (part) ->
  #           start_index = part * @top_page_size
  #           photos = Voting.get_top(@top_page_size, offset: start_index)
  #                    |> Enum.map(
  #                         fn (girl) -> %{url: girl.photo, caption: Girl.get_profile_url(girl)} end
  #                       )
  #           Messenger.send_photos(message.chat_id, photos)
  #           :timer.sleep(5000)
  #         end
  #       )
  #  end

  @spec handle_get_top_cmd(TextMessage.t) :: any
  defp handle_get_top_cmd(message) do
    optional_start_position = TextMessage.get_command_arg(message)
    offset = case Integer.parse(optional_start_position) do
      {start_position, ""} when start_position > 0 -> start_position - 1
      _ -> 0
    end
    send_girl_from_top(message.chat_id, offset)
  end

  @spec handle_help_cmd(TextMessage.t) :: any
  defp handle_help_cmd(message) do
    text = """
    Hi there! My mission is to find the most attractive girls on Instagram!
    Just select which of two girls looks better and vote by pressing a button below.
    I support following commands:

    /start - Get the next girls pair to compare.

    /showtop - Show the top girls in the competition. You can also pass a position to start showing from, i.e pass 10 to start from the 10th girl in the competition and skip the first 9.

    /help - Show this message.

    And also another type of commands, with an additional input:

    addgirl <link to one of her photos on instagram> - Add girl to the competition, You can add any girl, just paste a link of her instagram photo. The girl must have a public account. For example:
    addgirl https://www.instagram.com/p/BcPqz6sFMbb/

    girlinfo <girl username or link to the profile> - Get info about particular girl statistic in the competition. For example:
    girlinfo https://www.instagram.com/svetabily/
    girlinfo svetabily

    Some notes:
    In group chats you have to mention me (with the '@' sign) in the message, if you want to command me. In the private chat no mention needed.
    """
    Messenger.send_text(message.chat_id, text, disable_web_page_preview: true)
  end

  @spec handle_get_girl_info_cmd(TextMessage.t) :: any
  defp handle_get_girl_info_cmd(message) do
    girl_link = TextMessage.get_command_arg(message)
    case Voting.get_girl(girl_link) do
      {:ok, girl} -> display_girl_info(message.chat_id, girl)
      {:error, error_msg} -> Messenger.send_text(message.chat_id, error_msg)
    end
  end

  @spec display_girl_info(integer, Girl.t) :: any
  defp display_girl_info(chat_id, girl) do
    profile_url = Girl.get_profile_url(girl)
    Messenger.send_markdown(chat_id, "[#{girl.username}](#{profile_url})")
    Messenger.send_photo(chat_id, girl.photo)
    girl_position = Girl.get_position(girl)
    text = """
    Position in the competition: #{girl_position}
    Number of wins: #{girl.wins}
    Number of loses: #{girl.loses}
    """
    Messenger.send_text(chat_id, text)
  end

  @spec on_callback(Callback.t) :: any
  defp on_callback(message) do
    Logger.info("Process callback #{inspect message}")
    callbacks = %{
      @vote_callback => &handle_vote_callback/1,
      @get_top_callback => &handle_get_top_callback/1,
    }
    callback_name = Callback.get_name(message)
    handler = callbacks[callback_name]
    if handler do
      Logger.info("handle #{callback_name} callback")
      handler.(message)
    else
      Logger.warn("Unknown callback: #{callback_name}")
    end
  end

  @spec handle_vote_callback(Callback.t) :: any
  defp handle_vote_callback(message) do
    callback_args = Callback.get_args(message)
    [winner_username, loser_username] = String.split(callback_args, @usernames_separator)
    voters_group_id = build_voters_group_id(message.chat_id)
    voter_id = build_voter_id(message.user_id)
    case Voting.vote(voters_group_id, voter_id, winner_username, loser_username) do
      :ok ->
        task = Task.async(
          fn -> Messenger.send_notification(message.callback_id, "Vote for #{winner_username}")end
        )
        send_next_girls_pair(message.chat_id)
        Task.await(task)
      {:error, error} ->
        Messenger.send_notification(message.callback_id, "You already voted")
        Logger.warn("Can't vote: #{error}")
    end
  end

  @spec handle_get_top_callback(Callback.t) :: any
  defp handle_get_top_callback(message) do
    #    Messenger.delete_keyboard(message.chat_id, message.parent_msg_id)
    callback_args = Callback.get_args(message)
    girl_offset = case Integer.parse(callback_args) do
      {offset, ""} -> offset
      _ -> raise "Non-int arg for get top callback: #{callback_args}"
    end
    send_girl_from_top(message.chat_id, girl_offset)
    Messenger.answer_callback(message.callback_id)
  end

  @spec send_girl_from_top(integer, integer) :: any
  defp send_girl_from_top(chat_id, girl_offset) do
    case Voting.get_top(2, offset: girl_offset) do
      [current_girl | next_girls] ->
        keyboard = if length(next_girls) != 0 do
          next_girl_offset = Integer.to_string(girl_offset + 1)
          [[%{text: "Next", payload: Callback.build_payload(@get_top_callback, next_girl_offset)}]]
        else
          nil
        end
        Messenger.send_photo(
          chat_id,
          current_girl.photo,
          caption: "#{girl_offset + 1}th place: " <> Girl.get_profile_url(current_girl),
          keyboard: keyboard
        )
      [] -> Logger.warn("Girl offset #{girl_offset} more than number of girls in the competition")
    end
  end
end
