tinychain chat
==============

One of the most annoying parts of blockchains is addresses. Aside from phone numbers, people don't remember long strings of unique ID's. But what if there was a better way? 

I was dwelling on the old designs of Bitmessage and IRC. IRC was the first internet chat protocol to really take hold, and Bitmessage was a more modern cryptographic take on internet text chat where users had public key identities. 

What if we had a blockchain that came with a chat function? 

The original bitcoin client actually allowed users to send bitcoins to IP addresses. This is obviously insecure but is a fascinating way to think about alternative paths in the idea maze.

## The idea.

Tinychain CLI has a built in chat. 

`tinychain chat` shows you a list of your chats.

`tinychain chat tinychat:group:123123123123` will connect you to a group chat, where `123123123123` is the unique secret key that allows you to gain access to the chat.

From there, you can type and send messages, and they are signed by your private key, and encrypted by the chat key.

**What's amazing** is that you can use the chat to provide a better UX to addresses. 

Users choose usernames/nicks/handles whatever you want to call them. Each username/nick/handle is unique. The chat is a P2P state machine where there are a few transitions:

 - **post_message**. Message is (version, text, extradata, sig), encrypted_message is (payload = enc(payload, chat_key))
 - **set_username**. SetUsernameMessage is (version, username, sig, pubkey). 
   - the state machine dictates that usernames must be unique. so there is no risk of duplicates.
   - usernames are forced to be composed of characters `a-zA-Z0-9_.`
 - **set_topic**. sets the chat name of the group chat.

Users can write `/send davo 20 tiny` to send 20 TINYCOIN to davo's public key.

The UX is really nice. Because you're sending a payment within the context of the group chat, you're implicitly trusting that the usernames are being used for...chatting. And so you can use them to send payments. 

I imagine this will be a fun experiment in UX:

```sh
$ tinychain chat newgroup
tinychat:group:123123123123

$ tinychain chat tinychat:group:123123123123
davo: hey whats' up
liam: yeah just chatting on a p2p chat
```