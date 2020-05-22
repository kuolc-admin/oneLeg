local ResultCell(result) = {
    "type": "box",
    "layout": "vertical",
    "contents": [
        {
            "type": "text",
            "text": result.option,
            "align": "start",
            "size": "sm",
            "gravity": "center",
            "margin": "sm",
            "wrap": false
        },
        {
            "type": "text",
            "text": result.answerersText,
            "size": "xs",
            "color": "#999999",
            "wrap": true
        },
        {
            "type": "box",
            "layout": "horizontal",
            "contents": [
                {
                    "type": "box",
                    "layout": "vertical",
                    "contents": [
                        {
                            "type": "box",
                            "layout": "vertical",
                            "contents": [
                                {
                                    "type": "filler"
                                }
                            ],
                            "width": result.rate + "%",
                            "backgroundColor": if result.rate > 0
                                then if result.isMajority then "#67C47A" else "#CCCCCC"
                                else "#FFFFFF",
                            "height": "18px"
                        }
                    ],
                    "width": "80%",
                    "spacing": "none"
                },
                {
                    "type": "text",
                    "text": result.count + "人",
                    "size": "sm",
                    "gravity": "top",
                    "margin": "lg"
                }
            ],
            "margin": "md"
        }
    ],
    "margin": "md"
};

local CommentCell(comment) = {
    "type": "box",
    "layout": "vertical",
    "contents": [
        {
            "type": "text",
            "text": "@" + comment.userName,
            "size": "sm"
        },
        {
            "type": "text",
            "text": comment.text,
            "wrap": true,
            "size": "sm"
        }
    ],
    "margin": "lg"
};

local args = std.parseJson(std.extVar("args"));

(if args.imageURL != "" then {
    "hero": {
        "type": "image",
        "url": args.imageURL,
        "size": "full",
        "aspectRatio": args.imageAspectRatio,
        "aspectMode": "cover"
    }
} else {}) + {
    "type": "bubble",
    "size": "mega",
    "body": {
        "type": "box",
        "layout": "vertical",
        "contents": [
            {
                "type": "text",
                "text": "今日の1レッグ（解説）",
                "weight": "bold",
                "size": "xl"
            }
        ] + (if args.text != "" then [
            {
                "type": "text",
                "text": args.text,
                "margin": "md",
                "wrap": true
            }
        ] else []) + [
            {
                "type": "box",
                "layout": "vertical",
                "contents": [
                    {
                        "type": "text",
                        "text": "回答結果（計" + args.count + "人）",
                        "size": "lg",
                        "weight": "bold"
                    },
                    {
                        "type": "separator",
                        "margin": "sm"
                    }
                ] + [
                    ResultCell(result) for result in args.results
                ],
                "margin": "lg"
            },
            {
                "type": "box",
                "layout": "vertical",
                "contents": [
                    {
                        "type": "text",
                        "text": "コメント",
                        "size": "lg",
                        "weight": "bold"
                    },
                    {
                        "type": "separator",
                        "margin": "sm"
                    }
                ] + std.flattenArrays([
                    (
                        (if std.length(args.commentsDict[option]) > 0 then [
                            {
                                "type": "text",
                                "text": option,
                                "margin": "lg",
                                "size": "md",
                                "weight": "bold"
                            },
                        ] else []) + [
                            CommentCell(comment) for comment in args.commentsDict[option]
                        ]
                    ) for option in std.objectFields(args.commentsDict)
                ]),
                "margin": "xxl"
            }
        ]
    }
}
