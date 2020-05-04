local args = std.parseJson(std.extVar("args"));

{
    "type": "bubble",
    "size": "mega",
    "hero": {
        "type": "image",
        "url": args.imageURL,
        "size": "full",
        "aspectRatio": args.imageAspectRatio,
        "aspectMode": "cover"
    },
    "body": {
        "type": "box",
        "layout": "vertical",
        "contents": [
            {
                "type": "text",
                "text": "今日の1レッグ",
                "weight": "bold",
                "size": "xl",
            }
        ] + (if args.text != "" then [
            {
                "type": "text",
                "text": args.text,
                "margin": "md",
            }
        ] else []) + [
            {
                "type": "box",
                "layout": "baseline",
                "margin": "md",
                "contents": [
                    {
                        "type": "text",
                        "text": "難易度",
                        "size": "sm",
                        "color": "#999999",
                        "margin": "md",
                        "flex": 0,
                    }
                ] + [
                    {
                        "type": "icon",
                        "size": "sm",
                        "url": "https://scdn.line-apps.com/n/channel_devcenter/img/fx/review_gold_star_28.png",
                        "margin": if i == 0 then "md" else "none",
                    } for i in std.range(0, args.difficulty-1) 
                ] + [
                    {
                        "type": "icon",
                        "size": "sm",
                        "url": "https://scdn.line-apps.com/n/channel_devcenter/img/fx/review_gray_star_28.png",
                    } for i in std.range(args.difficulty, 4)                    
                ]
            },
            {
                "type": "box",
                "layout": "vertical",
                "margin": "lg",
                "spacing": "sm",
                "contents": [
                    {
                        "type": "box",
                        "layout": "baseline",
                        "spacing": "sm",
                        "contents": [
                            {
                                "type": "text",
                                "text": "出題者",
                                "color": "#aaaaaa",
                                "size": "sm",
                                "flex": 0
                            }
                        ] + (if args.setter != "" then [
                            {
                                "type": "text",
                                "text": args.setter,
                                "wrap": true,
                                "color": "#666666",
                                "size": "sm",
                                "flex": 0,
                                "margin": "md"
                            }
                        ] else [])
                    }
                ]
            }
        ]
    },
    "footer": {
        "type": "box",
        "layout": "vertical",
        "spacing": "sm",
        "contents": [
            {
                "type": "button",
                "style": "link",
                "height": "sm",
                "action": {
                    "type":"uri",
                    "label":"回答する",
                    "uri":"https://liff.line.me/1654090449-62QRAB0Z"
                }
            }
        ],
        "flex": 0
    }
}